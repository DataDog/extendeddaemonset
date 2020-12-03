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
	namespace := testConfig.namespace
	ctx := context.Background()

	Context("Initial deployment", func() {
		name := "eds-foo"

		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		It("Should deploy EDS ", func() {
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

		It("Should auto-pause and auto-fail canary on restarts", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
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

			Expect(k8sClient.Update(ctx, eds)).Should(Succeed())

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused
			}), longTimeout, interval).
				Should(
					BeTrue(),
					func() string {
						return fmt.Sprintf(
							"EDS should be in [%s] state but is currently in [%s]",
							datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused,
							eds.Status.State,
						)
					},
				)

			Expect(eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey]).
				Should(
					Equal("true"),
					"EDS canary should be marked paused",
				)

			Eventually(withEDS(key, eds, func() bool {
				return (eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed)
			}), longTimeout, interval).
				Should(
					BeTrue(),
					func() string {
						return fmt.Sprintf(
							"EDS should be in [%s] state but is currently in [%s]",
							datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed,
							eds.Status.State,
						)
					},
				)

			Expect(eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryFailedAnnotationKey]).
				Should(
					Equal("true"),
					"EDS canary should be marked failed",
				)
		})

		It("Should delete EDS", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, eds)).Should(Succeed())

			pods := &corev1.PodList{}
			listOptions := []client.ListOption{
				client.InNamespace(namespace),
				client.MatchingLabels{
					datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey: name,
				},
			}
			Eventually(withList(listOptions, pods, "EDS pods", func() bool {
				return len(pods.Items) == 0
			}), timeout, interval).Should(BeTrue(), "All EDS pods should be destroyed")
		})
	})
})
