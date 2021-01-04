// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build !e2e

package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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

	intString10 := intstr.FromInt(10)
	reconcileFrequency := &metav1.Duration{Duration: time.Millisecond * 100}
	namespace := testConfig.namespace
	ctx := context.Background()

	Context("Using ExtendedDaemonsetSetting", func() {
		name := "eds-setting"
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		It("Should create one pod by nods", func() {
			edsOptions := &testutils.NewExtendedDaemonsetOptions{
				CanaryStrategy: nil,
				RollingUpdate: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
					MaxPodSchedulerFailure: &intString10,
					MaxUnavailable:         &intString10,
					MaxParallelPodCreation: datadoghqv1alpha1.NewInt32(20),
					SlowStartIntervalDuration: &metav1.Duration{
						Duration: 1 * time.Millisecond,
					},
					SlowStartAdditiveIncrease: &intString10,
				},
				ReconcileFrequency: reconcileFrequency,
			}
			eds := testutils.NewExtendedDaemonset(namespace, name, "k8s.gcr.io/pause:latest", edsOptions)
			Expect(k8sClient.Create(ctx, eds)).Should(Succeed())

			eds = &datadoghqv1alpha1.ExtendedDaemonSet{}
			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.ActiveReplicaSet != ""
			}), timeout, interval).Should(BeTrue(), func() string {
				return fmt.Sprintf(
					"ActiveReplicaSet should be set EDS: %#v",
					eds.Status,
				)
			},
			)

			ers := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}
			erskey := types.NamespacedName{
				Namespace: namespace,
				Name:      eds.Status.ActiveReplicaSet,
			}
			Eventually(withERS(erskey, ers, func() bool {
				return ers.Status.Desired == ers.Status.Current
			}), timeout, interval).Should(BeTrue(), func() string {
				return fmt.Sprintf(
					"ers.Status.Desired should be equal to ers.Status.Current, status: %#v",
					ers.Status,
				)
			},
			)
		})
	})
})
