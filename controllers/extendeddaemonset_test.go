// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

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
	"k8s.io/apimachinery/pkg/labels"
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

	Context("Initial deployment", func() {
		var err error
		namespace := "default"
		name := "eds-foo"
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		It("Should handle EDS ", func() {
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
			Expect(k8sClient.Create(context.Background(), eds)).Should(Succeed())

			eds = &datadoghqv1alpha1.ExtendedDaemonSet{}
			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				return eds.Status.ActiveReplicaSet != ""
			}, timeout, interval).Should(BeTrue())

			ers := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}
			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), types.NamespacedName{
					Namespace: namespace,
					Name:      eds.Status.ActiveReplicaSet,
				}, ers)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				nodeList := &corev1.NodeList{}
				err = k8sClient.List(context.Background(), nodeList)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return ers.Status.Status == "active" && int(ers.Status.Available) == len(nodeList.Items)
			}, timeout, interval).Should(BeTrue())
		})

		It("Should do canary deployment", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				b, _ := json.MarshalIndent(eds.Status, "", "  ")
				fmt.Fprintf(GinkgoWriter, string(b))

				eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("k8s.gcr.io/pause:3.1")

				err = k8sClient.Update(context.Background(), eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return eds.Status.Canary != nil && eds.Status.Canary.ReplicaSet != ""
			}, timeout, interval).Should(BeTrue())
		})

		It("Should add canary labels", func() {
			Eventually(func() bool {
				canaryPods := &corev1.PodList{}
				listOptions := []client.ListOption{
					client.MatchingLabels{
						datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelValue,
					},
				}

				err = k8sClient.List(context.Background(), canaryPods, listOptions...)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return len(canaryPods.Items) == 2
			}, timeout, interval).Should(BeTrue())
		})

		It("Should remove canary labels", func() {
			Eventually(func() bool {
				eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				if eds.Annotations == nil {
					eds.Annotations = make(map[string]string)
				}

				eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryValidAnnotationKey] = eds.Status.Canary.ReplicaSet

				if err = k8sClient.Update(context.Background(), eds); err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())

			Eventually(func() bool {
				eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				canaryPods := &corev1.PodList{}
				listOptions := []client.ListOption{
					client.MatchingLabels{
						datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelValue,
						datadoghqv1alpha1.ExtendedDaemonSetReplicaSetNameLabelKey:   eds.Status.ActiveReplicaSet,
					},
				}

				err = k8sClient.List(context.Background(), canaryPods, listOptions...)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return len(canaryPods.Items) == 0
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Using ExtendedDaemonsetSetting", func() {
		var err error
		namespace := "default"
		name := "eds-setting"
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		It("Should create node", func() {
			nodeWorker := testutils.NewNode("node-worker", map[string]string{
				"role": "eds-setting-worker",
			})
			Expect(k8sClient.Create(context.Background(), nodeWorker)).Should(Succeed())
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
			Expect(k8sClient.Create(context.Background(), edsNodeSetting)).Should(Succeed())

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
			Expect(k8sClient.Create(context.Background(), eds)).Should(Succeed())

			eds = &datadoghqv1alpha1.ExtendedDaemonSet{}
			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				return eds.Status.ActiveReplicaSet != ""
			}, timeout, interval).Should(BeTrue())

			podList := &corev1.PodList{}
			Eventually(func() bool {
				err = k8sClient.List(context.Background(), podList, &client.ListOptions{
					Namespace:     namespace,
					LabelSelector: labels.Set(map[string]string{"extendeddaemonset.datadoghq.com/name": name}).AsSelector(),
				})
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				return len(podList.Items) == 3
			}, timeout, interval).Should(BeTrue())

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
