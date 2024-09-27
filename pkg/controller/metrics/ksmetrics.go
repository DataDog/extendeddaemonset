package metrics

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	edsconfig "github.com/DataDog/extendeddaemonset/pkg/config"
)

// AddMetrics add given metricFamilies for type given in gvk.
func AddMetrics(gvk schema.GroupVersionKind, scheme *runtime.Scheme, h Handler, metricFamilies []generator.FamilyGenerator) error {
	restConfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	httpClient, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return err
	}
	serializerCodec := serializer.NewCodecFactory(scheme)
	paramCodec := runtime.NewParameterCodec(scheme)

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig, httpClient)
	if err != nil {
		return err
	}
	mapping, err := restMapper.RESTMapping(gvk.GroupKind())
	if err != nil {
		return err
	}

	restClient, err := apiutil.RESTClientForGVK(gvk, false, restConfig, serializerCodec, httpClient)
	if err != nil {
		return err
	}

	obj, err := scheme.New(gvk)
	if err != nil {
		return err
	}

	listGVK := gvk.GroupVersion().WithKind(gvk.Kind + "List")
	listObj, err := scheme.New(listGVK)
	if err != nil {
		return err
	}

	namespaces := edsconfig.GetWatchNamespaces()
	if len(namespaces) == 0 {
		namespaces = append(namespaces, "")
	}
	for _, ns := range namespaces {
		lw := &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				res := listObj.DeepCopyObject()
				localErr := restClient.Get().NamespaceIfScoped(ns, ns != "").Resource(mapping.Resource.Resource).VersionedParams(&opts, paramCodec).Do(context.Background()).Into(res)

				return res, localErr
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				opts.Watch = true

				return restClient.Get().NamespaceIfScoped(ns, ns != "").Resource(mapping.Resource.Resource).VersionedParams(&opts, paramCodec).Watch(context.Background())
			},
		}

		err = h.RegisterStore(metricFamilies, obj, lw)
		if err != nil {
			return err
		}
	}

	return nil
}

// newMetricsStore return new metrics store.
func newMetricsStore(generators []generator.FamilyGenerator, expectedType interface{}, lw cache.ListerWatcher) *metricsstore.MetricsStore {
	// Generate collector per namespace.
	composedMetricGenFuncs := generator.ComposeMetricGenFuncs(generators)
	headers := generator.ExtractMetricFamilyHeaders(generators)
	store := metricsstore.NewMetricsStore(headers, composedMetricGenFuncs)
	reflectorPerNamespace(context.TODO(), lw, expectedType, store)

	return store
}

func reflectorPerNamespace(
	ctx context.Context,
	lw cache.ListerWatcher,
	expectedType interface{},
	store cache.Store,
) {
	reflector := cache.NewReflector(lw, expectedType, store, 0)
	go reflector.Run(ctx.Done())
}
