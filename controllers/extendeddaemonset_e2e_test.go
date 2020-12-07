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
	. "github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/conditions"
	"github.com/DataDog/extendeddaemonset/controllers/testutils"
	// +kubebuilder:scaffold:imports
)

const (
	timeout     = 1 * time.Minute
	longTimeout = 5 * time.Minute
	interval    = 2 * time.Second
)

var (
	intString2  = intstr.FromInt(2)
	intString10 = intstr.FromInt(10)
	namespace   = testConfig.namespace
	ctx         = context.Background()
)

// These tests may take several minutes to run, check you go test timeout
var _ = Describe("ExtendedDaemonSet e2e updates and recovery", func() {
	Context("Initial deployment", func() {
		name := "eds-foo"

		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		nodeList := &corev1.NodeList{}

		It("Should deploy EDS", func() {
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
				"does-not-exist", // command that does not exist
			}

			Expect(k8sClient.Update(ctx, eds)).Should(Succeed())

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused
			}), longTimeout, interval).Should(
				BeTrue(),
				func() string {
					return fmt.Sprintf(
						"EDS should be in [%s] state but is currently in [%s]",
						datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused,
						eds.Status.State,
					)
				},
			)

			pauseValue := eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey]
			Expect(pauseValue).Should(
				Equal("true"),
				"EDS canary should be marked paused",
			)

			pauseReason := eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedReasonAnnotationKey]
			Expect(pauseReason).Should(Or(Equal("StartError")))

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed
			}), longTimeout, interval).Should(
				BeTrue(),
				func() string {
					return fmt.Sprintf(
						"EDS should be in [%s] state but is currently in [%s]",
						datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed,
						eds.Status.State,
					)
				},
			)

			Expect(eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryFailedAnnotationKey]).Should(
				Equal("true"),
				"EDS canary should be marked failed",
			)
		})

		It("Should recover from failed on update", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
			b, _ := json.MarshalIndent(eds.Status, "", "  ")
			fmt.Fprintf(GinkgoWriter, string(b))

			clearFailureAnnotations(eds)

			eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("gcr.io/google-containers/alpine-with-bash:1.0")
			eds.Spec.Template.Spec.Containers[0].Command = []string{
				"tail", "-f", "/dev/null",
			}
			Eventually(withUpdate(eds, "EDS")).Should(BeTrue())

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanary
			}), timeout, interval).Should(BeTrue())

			canaryReplicaSet := eds.Status.Canary.ReplicaSet
			eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryValidAnnotationKey] = canaryReplicaSet
			Eventually(withUpdate(eds, "EDS")).Should(BeTrue())

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateRunning
			}), timeout, interval).Should(BeTrue())
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

var _ = Describe("ExtendedDaemonSet e2e PodCannotStart condition", func() {
	name := "eds-foo-cannot-start"

	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	BeforeEach(func() {
		edsOptions := &testutils.NewExtendedDaemonsetOptions{
			CanaryStrategy: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
				Duration: &metav1.Duration{Duration: 1 * time.Minute},
				Replicas: &intString2,
				AutoPause: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
					Enabled:              datadoghqv1alpha1.NewBool(true),
					MaxSlowStartDuration: &metav1.Duration{Duration: 20 * time.Second},
				},
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
	})

	AfterEach(func() {
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

		edslist := &datadoghqv1alpha1.ExtendedDaemonSetList{}
		listOptions = []client.ListOption{
			client.InNamespace(namespace),
		}
		Eventually(withList(listOptions, edslist, "EDS instances", func() bool {
			return len(edslist.Items) == 0
		}), timeout, interval).Should(BeTrue(), "All EDS instances should be destroyed")
	})

	pauseOnCannotStart := func(configureEDS func(eds *datadoghqv1alpha1.ExtendedDaemonSet), expectedReasons ...string) {
		eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
		Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
		b, _ := json.MarshalIndent(eds.Status, "", "  ")
		fmt.Fprintf(GinkgoWriter, string(b))
		configureEDS(eds)

		Eventually(withUpdate(eds, "EDS")).Should(BeTrue())

		eds = &datadoghqv1alpha1.ExtendedDaemonSet{}
		Eventually(withEDS(key, eds, func() bool {
			return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused
		}), timeout, interval).Should(BeTrue())

		pausedValue := eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey]
		Expect(pausedValue).Should(Equal("true"), "EDS canary should be marked paused")

		pausedReason := eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedReasonAnnotationKey]

		var matchers []GomegaMatcher
		for _, reason := range expectedReasons {
			matchers = append(matchers, Equal(reason))
		}

		Expect(pausedReason).Should(Or(matchers...), "EDS canary should be paused with expected reason")

		ers := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}
		ersKey := types.NamespacedName{
			Namespace: namespace,
			Name:      eds.Status.Canary.ReplicaSet,
		}
		Expect(k8sClient.Get(ctx, ersKey, ers)).Should(Succeed())

		var cannotStartCondition *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCondition
		Eventually(withEDS(key, eds, func() bool {
			cannotStartCondition = conditions.GetExtendedDaemonSetReplicaSetStatusCondition(&ers.Status, datadoghqv1alpha1.ConditionTypePodCannotStart)
			return cannotStartCondition != nil
		}), timeout, interval).Should(BeTrue())

		Expect(cannotStartCondition.Status).Should(Equal(corev1.ConditionTrue))

		matchers = []GomegaMatcher{}
		for _, reason := range expectedReasons {
			matchers = append(matchers, MatchRegexp(fmt.Sprintf("Pod eds-foo-.*? cannot start with reason: %s", reason)))
		}
		Expect(cannotStartCondition.Message).Should(Or(matchers...))
	}

	Context("When pod has image pull error", func() {

		It("Should promptly auto-pause canary", func() {
			pauseOnCannotStart(func(eds *datadoghqv1alpha1.ExtendedDaemonSet) {
				eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("gcr.io/missing")
				eds.Spec.Template.Spec.Containers[0].Command = []string{
					"does-not-matter",
				}
			}, "ErrImagePull", "ImagePullBackOff")
		})
	})

	Context("When pod has container config error", func() {

		It("Should promptly auto-pause canary", func() {
			pauseOnCannotStart(func(eds *datadoghqv1alpha1.ExtendedDaemonSet) {
				eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("gcr.io/google-containers/alpine-with-bash:1.0")
				eds.Spec.Template.Spec.Containers[0].Command = []string{
					"tail", "-f", "/dev/null",
				}
				eds.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
					{
						Name: "missing",
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "missing",
								},
								Key: "missing",
							},
						},
					},
				}
			}, "CreateContainerConfigError")
		})
	})

	Context("When pod has missing volume", func() {

		It("Should promptly auto-pause canary", func() {
			pauseOnCannotStart(func(eds *datadoghqv1alpha1.ExtendedDaemonSet) {
				eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("gcr.io/google-containers/alpine-with-bash:1.0")
				eds.Spec.Template.Spec.Containers[0].Command = []string{
					"tail", "-f", "/dev/null",
				}
				eds.Spec.Template.Spec.Volumes = []corev1.Volume{
					{
						Name: "missing-config-map",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "missing",
								},
							},
						},
					},
				}
				eds.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
					{
						Name:      "missing-config-map",
						MountPath: "/etc/missing",
					},
				}
			}, "SlowStartTimeoutExceeded")
		})
	})
})

func withUpdate(obj runtime.Object, desc string) condFn {
	return func() bool {
		err := k8sClient.Update(context.Background(), obj)
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Failed to update %s: %v", desc, err)
			return false
		}
		return true
	}
}

func clearFailureAnnotations(eds *datadoghqv1alpha1.ExtendedDaemonSet) {
	delete(eds.Annotations, datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey)
	delete(eds.Annotations, datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedReasonAnnotationKey)
	delete(eds.Annotations, datadoghqv1alpha1.ExtendedDaemonSetCanaryFailedAnnotationKey)
	delete(eds.Annotations, datadoghqv1alpha1.ExtendedDaemonSetCanaryFailedReasonAnnotationKey)
}
