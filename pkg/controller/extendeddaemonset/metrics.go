// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonset

import (
	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/controller/metrics"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils"
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
	extendeddaemonsetCreated                        = "eds_created"
	extendeddaemonsetStatusDesired                  = "eds_status_desired"
	extendeddaemonsetStatusCurrent                  = "eds_status_current"
	extendeddaemonsetStatusReady                    = "eds_status_ready"
	extendeddaemonsetStatusAvailable                = "eds_status_available"
	extendeddaemonsetStatusUpToDate                 = "eds_status_uptodate"
	extendeddaemonsetStatusIgnoredUnresponsiveNodes = "eds_status_ignored_unresponsive_nodes"
	extendeddaemonsetStatusCanaryActivated          = "eds_status_canary_activated"
	extendeddaemonsetStatusCanaryNumberOfNodes      = "eds_status_canary_node_number"
	extendeddaemonsetLabels                         = "eds_labels"
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

				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				extraKeys, extraValues := utils.BuildInfoLabels(&eds.ObjectMeta)

				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       1,
							LabelKeys:   append(labelKeys, extraKeys...),
							LabelValues: append(labelValues, extraValues...),
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

				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)

				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.CreationTimestamp.Unix()),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Desired),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Current),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Ready),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.Available),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.UpToDate),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
						},
					},
				}
			},
		},
		{
			Name: extendeddaemonsetStatusIgnoredUnresponsiveNodes,
			Type: ksmetric.Gauge,
			Help: "The total number of nodes that are ignored by the rolling update strategy due to an unresponsive state",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				eds := obj.(*datadoghqv1alpha1.ExtendedDaemonSet)
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(eds.Status.IgnoredUnresponsiveNodes),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				val := float64(0)
				rs := ""
				if eds.Status.Canary != nil {
					val = 1
					rs = eds.Status.Canary.ReplicaSet
				}
				Labelkeys := append(labelKeys, "replicaset")
				Labelvalues := append(labelValues, rs)
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
				labelKeys, labelValues := utils.GetLabelsValues(&eds.ObjectMeta)
				val := float64(0)
				if eds.Status.Canary != nil {
					val = float64(len(eds.Status.Canary.Nodes))
				}
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       val,
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
						},
					},
				}
			},
		},
	}
}
