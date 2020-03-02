// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

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
	ersCreated                        = "ers_created"
	ersStatusDesired                  = "ers_status_desired"
	ersStatusCurrent                  = "ers_status_current"
	ersStatusReady                    = "ers_status_ready"
	ersStatusAvailable                = "ers_status_available"
	ersStatusIgnoredUnresponsiveNodes = "ers_status_ignored_unresponsive_nodes"
	ersLabels                         = "ers_labels"
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
				ers := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := utils.GetLabelsValues(&ers.ObjectMeta)
				extraKeys, extraValues := utils.BuildInfoLabels(&ers.ObjectMeta)
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
			Name: ersCreated,
			Type: ksmetric.Gauge,
			Help: "Unix creation timestamp",
			GenerateFunc: func(obj interface{}) *ksmetric.Family {
				ers := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := utils.GetLabelsValues(&ers.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(ers.CreationTimestamp.Unix()),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				ers := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := utils.GetLabelsValues(&ers.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(ers.Status.Desired),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				ers := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := utils.GetLabelsValues(&ers.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(ers.Status.Current),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				ers := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := utils.GetLabelsValues(&ers.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(ers.Status.Ready),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				ers := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := utils.GetLabelsValues(&ers.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(ers.Status.Available),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
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
				ers := obj.(*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet)
				labelKeys, labelValues := utils.GetLabelsValues(&ers.ObjectMeta)
				return &ksmetric.Family{
					Metrics: []*ksmetric.Metric{
						{
							Value:       float64(ers.Status.IgnoredUnresponsiveNodes),
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
						},
					},
				}
			},
		},
	}
}
