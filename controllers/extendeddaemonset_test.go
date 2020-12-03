// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build !e2e

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers/testutils"
	// +kubebuilder:scaffold:imports
)

// This test may take ~30s to run, check you go test timeout
var _ = Describe("ExtendedDaemonSet Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 2

	intString2 := intstr.FromInt(2)
	intString10 := intstr.FromInt(10)
	namespace := testConfig.namespace
	ctx := context.Background()

	Context("Initial deployment", func() {
		name := "eds-foo"
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		It("Should handle EDS ", func() {
			nodeList := &corev1.NodeList{}
			Expect(k8sClient.List(ctx, nodeList)).Should(Succeed())

			edsOptions := &testutils.NewExtendedDaemonsetOptions{
				CanaryStrategy: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					Duration: &metav1.Duration{Duration: 1 * time.Minute},
					Replicas: &intString2,
				},
				RollingUpdate: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
					MaxUnavailable:         &intString10,
					MaxParallelPodCreation: datadoghqv1alpha1.NewInt32(20),
				},
			}
			eds := testutils.NewExtendedDaemonset(namespace, name, "k8s.gcr.io/pause:latest", edsOptions)
			Expect(k8sClient.Create(ctx, eds)).Should(Succeed())

			eds = &datadoghqv1alpha1.ExtendedDaemonSet{}
			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.ActiveReplicaSet != ""
			}), timeout, interval).Should(BeTrue())

			ers := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}
			ersKey := types.NamespacedName{
				Namespace: namespace,
				Name:      eds.Status.ActiveReplicaSet,
			}
			Eventually(withERS(ersKey, ers, func() bool {
				return ers.Status.Status == "active" && int(ers.Status.Available) == len(nodeList.Items)
			}), timeout, interval).Should(BeTrue())
		})

		It("Should do canary deployment", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
			b, _ := json.MarshalIndent(eds.Status, "", "  ")
			fmt.Fprintf(GinkgoWriter, string(b))

			eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("k8s.gcr.io/pause:3.1")
			Expect(k8sClient.Update(ctx, eds)).Should(Succeed())

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.Canary != nil && eds.Status.Canary.ReplicaSet != ""
			}), timeout, interval).Should(BeTrue())
		})

		It("Should add canary labels", func() {
			canaryPods := &corev1.PodList{}
			listOptions := []client.ListOption{
				client.InNamespace(namespace),
				client.MatchingLabels{
					datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelValue,
				},
			}
			Eventually(withList(listOptions, canaryPods, "canary pods", func() bool {
				return len(canaryPods.Items) == 2
			}), timeout, interval).Should(BeTrue())
		})

		It("Should remove canary labels", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
			if eds.Annotations == nil {
				eds.Annotations = make(map[string]string)
			}

			canaryReplicaSet := eds.Status.Canary.ReplicaSet
			eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryValidAnnotationKey] = canaryReplicaSet
			Expect(k8sClient.Update(ctx, eds)).Should(Succeed())

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.ActiveReplicaSet == canaryReplicaSet
			}), timeout, interval).Should(BeTrue())

			canaryPods := &corev1.PodList{}
			listOptions := []client.ListOption{
				client.InNamespace(namespace),
				client.MatchingLabels{
					datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelValue,
					datadoghqv1alpha1.ExtendedDaemonSetReplicaSetNameLabelKey:   eds.Status.ActiveReplicaSet,
				},
			}
			Eventually(withList(listOptions, canaryPods, "canary pods", func() bool {
				return len(canaryPods.Items) == 0
			}), timeout, interval).Should(BeTrue())
		})
	})

	Context("Using ExtendedDaemonsetSetting", func() {
		// var err error
		name := "eds-setting"
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		It("Should create node", func() {
			nodeWorker := testutils.NewNode("node-worker", map[string]string{
				"role": "eds-setting-worker",
			})
			Expect(k8sClient.Create(ctx, nodeWorker)).Should(Succeed())
		})

		It("Should use DaemonsetSetting", func() {
			resouresRef := corev1.ResourceList{
				"cpu":    resource.MustParse("0.1"),
				"memory": resource.MustParse("20M"),
			}
			edsNodeSetting := testutils.NewExtendedDaemonsetSetting(namespace, "eds-setting-worker", name, &testutils.NewExtendedDaemonsetSettingOptions{
				Selector: map[string]string{"role": "eds-setting-worker"},
				Resources: map[string]corev1.ResourceRequirements{
					"main": {
						Requests: resouresRef,
					},
				},
			})
			Expect(k8sClient.Create(ctx, edsNodeSetting)).Should(Succeed())

			edsOptions := &testutils.NewExtendedDaemonsetOptions{
				CanaryStrategy: nil,
				RollingUpdate: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
					MaxPodSchedulerFailure: &intString10,
					MaxUnavailable:         &intString10,
					SlowStartIntervalDuration: &metav1.Duration{
						Duration: 1 * time.Millisecond,
					},
				},
			}
			eds := testutils.NewExtendedDaemonset(namespace, name, "k8s.gcr.io/pause:latest", edsOptions)
			Expect(k8sClient.Create(ctx, eds)).Should(Succeed())

			eds = &datadoghqv1alpha1.ExtendedDaemonSet{}
			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.ActiveReplicaSet != ""
			}), timeout, interval).Should(BeTrue())

			podList := &corev1.PodList{}
			listOptions := []client.ListOption{
				client.InNamespace(namespace),
				client.MatchingLabels{
					"extendeddaemonset.datadoghq.com/name": name,
				},
			}
			Eventually(withList(listOptions, podList, "pods", func() bool {
				return len(podList.Items) == 3
			}), timeout, interval).Should(BeTrue())

			// TODO: This loop below does not assert on anything in any way
			for _, pod := range podList.Items {
				if pod.Spec.NodeName == "node-worker" {
					for _, container := range pod.Spec.Containers {
						if container.Name != "main" {
							continue
						}
						if diff := cmp.Diff(resouresRef, container.Resources.Requests); diff != "" {
							fmt.Fprintf(GinkgoWriter, "diff pods resources: %s", diff)
						}
					}
				}
			}
		})
	})
})
