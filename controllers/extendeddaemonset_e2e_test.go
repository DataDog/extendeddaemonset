// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build e2e

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers/testutils"
	// +kubebuilder:scaffold:imports
)

// This test may take ~30s to run, check you go test timeout
var _ = Describe("ExtendedDaemonSet Controller", func() {
	const (
		timeout     = 1 * time.Minute
		longTimeout = 5 * time.Minute
		interval    = 2 * time.Second
	)

	intString2 := intstr.FromInt(2)
	intString10 := intstr.FromInt(10)

	Context("Initial deployment", func() {
		var err error

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
					fmt.Fprintf(GinkgoWriter, "Failed to get the eds instance: %v", err)
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
					client.InNamespace(namespace),
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
					client.InNamespace(namespace),
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

		It("Should auto-pause and auto-fail canary on restarts", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}
				b, _ := json.MarshalIndent(eds.Status, "", "  ")
				fmt.Fprintf(GinkgoWriter, string(b))
				eds.Spec.Strategy.Canary = &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					Duration:           &metav1.Duration{Duration: 1 * time.Minute},
					Replicas:           &intString2,
					NoRestartsDuration: &metav1.Duration{Duration: 1 * time.Minute},
					AutoPause: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     datadoghqv1alpha1.NewBool(true),
						MaxRestarts: datadoghqv1alpha1.NewInt32(2),
					},
					AutoFail: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     datadoghqv1alpha1.NewBool(true),
						MaxRestarts: datadoghqv1alpha1.NewInt32(3),
					},
				}
				eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("gcr.io/google-containers/alpine-with-bash:1.0")
				eds.Spec.Template.Spec.Containers[0].Command = []string{
					"foo",
				}

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

				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused
			}, longTimeout, interval).
				Should(
					BeTrue(),
					fmt.Sprintf(
						"EDS should be in [%s] state but is currently in [%s]",
						datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused,
						eds.Status.State,
					),
				)

			Expect(eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey]).
				Should(
					Equal("true"),
					"EDS canary should be marked paused",
				)

			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), key, eds)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return (eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed)
			}, longTimeout, interval).
				Should(
					BeTrue(),
					fmt.Sprintf(
						"EDS should be in [%s] state but is currently in [%s]",
						datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed,
						eds.Status.State,
					),
				)

			Expect(eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryFailedAnnotationKey]).
				Should(
					Equal("true"),
					"EDS canary should be marked failed",
				)

		})

		It("Should delete EDS", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(context.Background(), key, eds)).Should(Succeed())
			Expect(k8sClient.Delete(context.Background(), eds)).Should(Succeed())

			Eventually(func() bool {
				pods := &corev1.PodList{}
				listOptions := []client.ListOption{
					client.InNamespace(namespace),
					client.MatchingLabels{
						datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey: name,
					},
				}

				err = k8sClient.List(context.Background(), pods, listOptions...)
				if err != nil {
					fmt.Fprint(GinkgoWriter, err)
					return false
				}

				return len(pods.Items) == 0
			}, timeout, interval).Should(BeTrue(), "All EDS pods should be destroyed")
		})
	})
})
