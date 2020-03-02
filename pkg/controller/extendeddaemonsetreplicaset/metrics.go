// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

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

	ersCreated                        = "ers_created"
	ersStatusDesired                  = "ers_status_desired"
	ersStatusCurrent                  = "ers_status_current"
	ersStatusReady                    = "ers_status_ready"
	ersStatusAvailable                = "ers_status_available"
	ersStatusIgnoredUnresponsiveNodes = "ers_status_ignored_unresponsive_nodes"
	ersLabels                         = "ers_labels"
)

var (
	defaultLabelKeys = []string{
		resourceNamespacePromLabel,
		resourceNamePromLabel,
	}
)

// AddMetrics add ExtentedDaemonSetReplicaset metrics
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
			return clientset.DatadoghqV1alpha1().ExtendedDaemonSetReplicaSets(namespace).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return clientset.DatadoghqV1alpha1().ExtendedDaemonSetReplicaSets(namespace).Watch(opts)
		},
	}

	return h.RegisterStore(families, &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}, lw)
}

func generateMetricFamilies() []ksmetric.FamilyGenerator {
	return []ksmetric.FamilyGenerator{
		{
			Name: ersLabels,
			Type: ksmetric.Gauge,
			Help: "Kubernetes labels converted to Prometheus labels",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := buildERSInfoLabels(eds)
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
			Name: ersCreated,
			Type: ksmetric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
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
			Name: ersStatusDesired,
			Type: ksmetric.Gauge,
			Help: "The number of nodes that should be running the daemon pod.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
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
			Name: ersStatusCurrent,
			Type: ksmetric.Gauge,
			Help: "The number of nodes running at least one daemon pod and are supposed to.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
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
			Name: ersStatusReady,
			Type: ksmetric.Gauge,
			Help: "The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
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
			Name: ersStatusAvailable,
			Type: ksmetric.Gauge,
			Help: "The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and available.",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
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
			Name: ersStatusIgnoredUnresponsiveNodes,
			Type: ksmetric.Gauge,
			Help: "The total number of nodes that are ignored by the rolling update strategy due to an unresponsive state",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
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
	}
}

func getLabelsValues(eds *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) []string {
	return []string{eds.GetNamespace(), eds.GetName()}
}

var (
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func buildERSInfoLabels(ers *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) ([]string, []string) {
	labelKeys := make([]string, len(ers.Labels))
	for key := range ers.Labels {
		labelKeys = append(labelKeys, sanitizeLabelName(key))
	}
	sort.Strings(labelKeys)

	labelValues := make([]string, len(ers.Labels))
	for _, key := range labelKeys {
		labelValues = append(labelValues, ers.Labels[key])
	}

	labelKeys = append(defaultLabelKeys, labelKeys...)
	labelValues = append(getLabelsValues(ers), labelValues...)

	return labelKeys, labelValues
}

func sanitizeLabelName(s string) string {
	return invalidLabelCharRE.ReplaceAllString(s, "_")
}
