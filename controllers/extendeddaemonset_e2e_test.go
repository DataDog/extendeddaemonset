// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build e2e

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
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
	edsconditions "github.com/DataDog/extendeddaemonset/controllers/extendeddaemonset/conditions"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/conditions"
	"github.com/DataDog/extendeddaemonset/controllers/testutils"
	// +kubebuilder:scaffold:imports
)

const (
	timeout     = 1 * time.Minute
	longTimeout = 5 * time.Minute
	interval    = 2 * time.Second

	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Purple = "\033[35m"
	Bold   = "\x1b[1m"
)

var (
	intString1  = intstr.FromInt(1)
	intString2  = intstr.FromInt(2)
	intString10 = intstr.FromInt(10)
	namespace   = testConfig.namespace
	ctx         = context.Background()
)

func logPreamble() string {
	return Bold + "E2E >> " + Reset
}

func info(format string, a ...interface{}) {
	ginkgoLog(logPreamble()+Purple+Bold+format+Reset, a...)
}

func warn(format string, a ...interface{}) {
	ginkgoLog(logPreamble()+Red+Bold+format+Reset, a...)
}

func ginkgoLog(format string, a ...interface{}) {
	fmt.Fprintf(GinkgoWriter, format, a...)
}

// These tests may take several minutes to run, check you go test timeout
var _ = Describe("ExtendedDaemonSet e2e updates and recovery", func() {
	Context("Initial deployment", func() {
		name := "eds-fail"

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
			info("EDS status:\n%s\n", spew.Sdump(eds.Status))

			eds.Spec.Strategy.Canary = &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
				Duration:           &metav1.Duration{Duration: 1 * time.Minute},
				Replicas:           &intString1,
				NoRestartsDuration: &metav1.Duration{Duration: 1 * time.Minute},
				AutoPause: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
					Enabled:     datadoghqv1alpha1.NewBool(true),
					MaxRestarts: datadoghqv1alpha1.NewInt32(1),
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
			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.Reason == "StartError"
			}), timeout, interval).Should(BeTrue(), func() string {
				return fmt.Sprintf(
					"EDS should be in [%s] state reason but is currently in [%s]",
					"StartError",
					eds.Status.Reason,
				)
			})

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateRunning
			}), longTimeout, interval).Should(
				BeTrue(),
				func() string {
					return fmt.Sprintf(
						"EDS should be in [%s] state but is currently in [%s]",
						datadoghqv1alpha1.ExtendedDaemonSetStatusStateRunning,
						eds.Status.State,
					)
				},
			)

			Eventually(withEDS(key, eds, func() bool {
				if edsconditions.GetExtendedDaemonSetStatusCondition(&eds.Status, datadoghqv1alpha1.ConditionTypeEDSCanaryFailed) != nil {
					return true
				}
				return false
			}), longTimeout, interval).ShouldNot(
				BeNil(),
				func() string {
					return fmt.Sprintf(
						"EDS canary failure should be present in the EDS.Status.Conditions: %v",
						eds.Status.Conditions,
					)
				},
			)
		})

		It("Should recover from failed on update", func() {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
			info("EDS status:\n%s\n", spew.Sdump(eds.Status))

			clearFailureAnnotations(eds)

			eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("gcr.io/google-containers/alpine-with-bash:1.0")
			eds.Spec.Template.Spec.Containers[0].Command = []string{
				"tail", "-f", "/dev/null",
			}
			Eventually(withUpdate(eds, "EDS"),
				timeout, interval).Should(BeTrue())

			Eventually(withEDS(key, eds, func() bool {
				return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanary
			}), timeout, interval).Should(BeTrue())

			canaryReplicaSet := eds.Status.Canary.ReplicaSet
			if eds.Annotations == nil {
				eds.Annotations = make(map[string]string)
			}
			eds.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryValidAnnotationKey] = canaryReplicaSet
			Eventually(withUpdate(eds, "EDS"),
				timeout, interval).Should(BeTrue())

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

	var (
		name string
		key  types.NamespacedName
	)

	BeforeEach(func() {
		name = fmt.Sprintf("eds-foo-%d", time.Now().Unix())
		key = types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}

		info("BeforeEach: Creating EDS %s\n", name)

		edsOptions := &testutils.NewExtendedDaemonsetOptions{
			CanaryStrategy: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
				Duration: &metav1.Duration{Duration: 1 * time.Minute},
				Replicas: &intString2,
				AutoPause: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
					Enabled:              datadoghqv1alpha1.NewBool(true),
					MaxSlowStartDuration: &metav1.Duration{Duration: 10 * time.Second},
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

		info("BeforeEach: Done creating EDS %s - active replicaset: %s\n", name, eds.Status.ActiveReplicaSet)
	})

	AfterEach(func() {
		info("AfterEach: Destroying EDS %s\n", name)
		eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
		Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
		info("AfterEach: Destroying EDS %s - canary replicaset: %s\n", name, eds.Status.Canary.ReplicaSet)
		info("AfterEach: Destroying EDS %s - active replicaset: %s\n", name, eds.Status.ActiveReplicaSet)

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

		erslist := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSetList{}
		Eventually(withList(listOptions, erslist, "ERS instances", func() bool {
			return len(erslist.Items) == 0
		}), timeout, interval).Should(BeTrue(), "All ERS instances should be destroyed")

		info("AfterEach: Done destroying EDS %s\n", name)
	})

	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
			Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
			warn("%s - FAILED: EDS status:\n%s\n\n", CurrentGinkgoTestDescription().TestText, spew.Sdump(eds.Status))
		}
	})

	pauseOnCannotStart := func(configureEDS func(eds *datadoghqv1alpha1.ExtendedDaemonSet), expectedReasons ...string) {
		eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
		Expect(k8sClient.Get(ctx, key, eds)).Should(Succeed())
		info("%s: %s - active replicaset: %s\n",
			CurrentGinkgoTestDescription().TestText,
			name, eds.Status.ActiveReplicaSet,
		)

		info("EDS status:\n%s\n", spew.Sdump(eds.Status))
		configureEDS(eds)

		info("EDS %s - updating\n", name)
		Eventually(withUpdate(eds, "EDS"),
			timeout, interval).Should(BeTrue())

		info("EDS %s - waiting for canary to be paused\n", name)
		eds = &datadoghqv1alpha1.ExtendedDaemonSet{}
		Eventually(withEDS(key, eds, func() bool {
			return eds.Status.State == datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused
		}), timeout, interval).Should(BeTrue())

		info("EDS status:\n%s\n", spew.Sdump(eds.Status))

		cond := edsconditions.GetExtendedDaemonSetStatusCondition(&eds.Status, datadoghqv1alpha1.ConditionTypeEDSCanaryPaused)
		Expect(cond).ShouldNot(BeNil())
		Expect(cond.Status).Should(Equal(corev1.ConditionTrue))
		pausedReason := cond.Reason
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

		var cannotStartCondition *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCondition
		Eventually(withERS(ersKey, ers, func() bool {
			cannotStartCondition = conditions.GetExtendedDaemonSetReplicaSetStatusCondition(&ers.Status, datadoghqv1alpha1.ConditionTypePodCannotStart)
			return cannotStartCondition != nil
		}), timeout, interval).Should(BeTrue())

		Expect(cannotStartCondition.Status).Should(Equal(corev1.ConditionTrue))

		matchers = []GomegaMatcher{}
		for _, reason := range expectedReasons {
			matchers = append(matchers, MatchRegexp(fmt.Sprintf("Pod eds-foo-.*? cannot start with reason: %s", reason)))
		}
		Expect(cannotStartCondition.Message).Should(Or(matchers...))
		info("%s: %s - done\n",
			CurrentGinkgoTestDescription().TestText,
			name,
		)
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

// These tests may take several minutes to run, check you go test timeout
var _ = Describe("ExtendedDaemonSet e2e successful canary deployment update", func() {
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
					Replicas: &intString1,
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
			fmt.Fprintf(GinkgoWriter, "EDS status:\n%s\n", spew.Sdump(eds.Status))

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
				fmt.Fprintf(GinkgoWriter, "canary pods nb: %d ", len(canaryPods.Items))
				return len(canaryPods.Items) == 1
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

func withUpdate(obj runtime.Object, desc string) condFn {
	return func() bool {
		err := k8sClient.Update(context.Background(), obj)
		if err != nil {
			warn("Failed to update %s: %v\n", desc, err)
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
