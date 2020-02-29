package metrics

import (
	"context"

	ksmetric "k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"

	"k8s.io/client-go/tools/cache"
)

// newMetricsStore return new metrics store
func newMetricsStore(generators []ksmetric.FamilyGenerator, expectedType interface{}, lw cache.ListerWatcher) *metricsstore.MetricsStore {

	// Generate collector per namespace.
	composedMetricGenFuncs := ksmetric.ComposeMetricGenFuncs(generators)
	headers := ksmetric.ExtractMetricFamilyHeaders(generators)
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
