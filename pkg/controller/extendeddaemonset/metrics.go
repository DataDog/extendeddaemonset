// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonset

import (
	"regexp"
	"sort"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/controller/metrics"
	"github.com/datadog/extendeddaemonset/pkg/generated/clientset/versioned"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	ksmetric "k8s.io/kube-state-metrics/pkg/metric"
)

const (
	resourceNamePromLabel      = "name"
	resourceNamespacePromLabel = "namespace"

	extendeddaemonsetCreated                        = "extendeddaemonset_created"
	extendeddaemonsetStatusDesired                  = "extendeddaemonset_status_desired"
	extendeddaemonsetStatusCurrent                  = "extendeddaemonset_status_current"
	extendeddaemonsetStatusReady                    = "extendeddaemonset_status_ready"
	extendeddaemonsetStatusAvailable                = "extendeddaemonset_status_available"
	extendeddaemonsetStatusUpToDate                 = "extendeddaemonset_status_uptodate"
	extendeddaemonsetStatusIgnoredUnresponsiveNodes = "extendeddaemonset_status_ignored_unresponsive_nodes"
	extendeddaemonsetStatusCanaryActivated          = "extendeddaemonset_status_canary_activated"
	extendeddaemonsetStatusCanaryNumberOfNodes      = "extendeddaemonset_status_canary_node_number"
	extendeddaemonsetLabels                         = "extendeddaemonset_labels"
)

var (
	defaultLabelKeys = []string{
		resourceNamespacePromLabel,
		resourceNamePromLabel,
	}
)

// AddMetrics add ExtentedDaemonSet metrics
func AddMetrics(mgr manager.Manager, h metrics.Handler) error {
	families := generateMetricFamilies()

	clientset, err := versioned.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return err
	}

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return clientset.DatadoghqV1alpha1().ExtendedDaemonSets(namespace).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return clientset.DatadoghqV1alpha1().ExtendedDaemonSets(namespace).Watch(opts)
		},
	}

	return h.RegisterStore(families, &datadoghqv1alpha1.ExtendedDaemonSet{}, lw)
}

func generateMetricFamilies() []ksmetric.FamilyGenerator {
	return []ksmetric.FamilyGenerator{
		{
			Name: extendeddaemonsetLabels,
			Type: ksmetric.Gauge,
			Help: "Kubernetes labels converted to Prometheus labels",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				labelKeys, labelValues := buildEDSInfoLabels(eds)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       1,
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetCreated,
			Type: ksmetric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.CreationTimestamp.Unix()),
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetStatusDesired,
			Type: ksmetric.Gauge,
			Help: "The number of nodes that should be running the daemon pod.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Desired),
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetStatusCurrent,
			Type: ksmetric.Gauge,
			Help: "The number of nodes running at least one daemon pod and are supposed to.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Current),
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetStatusReady,
			Type: ksmetric.Gauge,
			Help: "The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Ready),
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetStatusAvailable,
			Type: ksmetric.Gauge,
			Help: "The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and available.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Available),
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetStatusUpToDate,
			Type: ksmetric.Gauge,
			Help: "The total number of nodes that are running updated daemon pod",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.UpToDate),
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetStatusIgnoredUnresponsiveNodes,
			Type: ksmetric.Gauge,
			Help: "The total number of nodes that are ignore by the rolling update strategy due to an unresponsive state",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.IgnoredUnresponsiveNodes),
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},

		{
			Name: extendeddaemonsetStatusCanaryActivated,
			Type: ksmetric.Gauge,
			Help: "The status of the canary deployement, set to 1 if active, else 0",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				val := float64(0)
				rs := ""
				if eds.Status.Canary != nil {
					val = 1
					rs = eds.Status.Canary.ReplicaSet
				}
				Labelkeys := append(defaultLabelKeys, "replicaset")
				Labelvalues := append(getLabelsValues(eds), rs)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       val,
							LabelKeys:   Labelkeys,
							LabelValues: Labelvalues,
						},
					},
				}
			},
		},

		{
			Name: extendeddaemonsetStatusCanaryNumberOfNodes,
			Type: ksmetric.Gauge,
			Help: "The total number of nodes that are running a canary pod",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				val := float64(0)
				if eds.Status.Canary != nil {
					val = float64(len(eds.Status.Canary.Nodes))
				}
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       val,
							LabelKeys:   defaultLabelKeys,
							LabelValues: getLabelsValues(eds),
						},
					},
				}
			},
		},
	}
}

func getLabelsValues(eds *datadoghqv1alpha1.ExtendedDaemonSet) []string {
	return []string{eds.GetNamespace(), eds.GetName()}
}

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func buildEDSInfoLabels(eds *datadoghqv1alpha1.ExtendedDaemonSet) ([]string, []string) {
	labelKeys := make([]string, len(eds.Labels))
	for key := range eds.Labels {
		labelKeys = append(labelKeys, sanitizeLabelName(key))
	}
	sort.Strings(labelKeys)

	labelValues := make([]string, len(eds.Labels))
	for _, key := range labelKeys {
		labelValues = append(labelValues, eds.Labels[key])
	}

	labelKeys = append(defaultLabelKeys, labelKeys...)
	labelValues = append(getLabelsValues(eds), labelValues...)

	return labelKeys, labelValues
}

func sanitizeLabelName(s string) string {
	return invalidLabelCharRE.ReplaceAllString(s, "_")
}
