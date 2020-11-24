// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
	eds "github.com/DataDog/extendeddaemonset/controllers/extendeddaemonset"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/conditions"
	podUtils "github.com/DataDog/extendeddaemonset/pkg/controller/utils/pod"
)

// ManageCanaryDeployment used to manage ReplicaSet in Canary state
func ManageCanaryDeployment(client client.Client, daemonset *v1alpha1.ExtendedDaemonSet, params *Parameters) (*Result, error) {
	result := &Result{}

	now := time.Now()
	metaNow := metav1.NewTime(now)
	var desiredPods, currentPods, availablePods, readyPods int32

	var needRequeue bool
	var err error
	isPaused, _ := eds.IsCanaryDeploymentPaused(daemonset.GetAnnotations())
	autoPauseEnabled := *daemonset.Spec.Strategy.Canary.AutoPause.Enabled
	autoPauseMaxRestarts := int(*daemonset.Spec.Strategy.Canary.AutoPause.MaxRestarts)

	isFailed := eds.IsCanaryDeploymentFailed(daemonset.GetAnnotations())
	autoFailEnabled := *daemonset.Spec.Strategy.Canary.AutoFail.Enabled
	autoFailMaxRestarts := int(*daemonset.Spec.Strategy.Canary.AutoFail.MaxRestarts)

	var lastRestartTime, newRestartTime time.Time
	var restartingPod string

	restartCondition := conditions.GetExtendedDaemonSetReplicaSetStatusCondition(params.NewStatus, v1alpha1.ConditionTypePodRestarting)
	if restartCondition != nil {
		lastRestartTime = restartCondition.LastUpdateTime.Time
	}

	// Canary mode
	for _, nodeName := range params.CanaryNodes {
		node := params.NodeByName[nodeName]
		desiredPods++
		if pod, ok := params.PodByNodeName[node]; ok {
			if pod == nil {
				result.PodsToCreate = append(result.PodsToCreate, node)
				needRequeue = true
			} else {
				if err = addPodLabel(client, pod, v1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey, v1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelValue); err != nil {
					params.Logger.Error(err, fmt.Sprintf("Couldn't add the canary label for pod '%s/%s', will retry later", pod.GetNamespace(), pod.GetName()))
					needRequeue = true
				}
				if pod.DeletionTimestamp != nil {
					needRequeue = true
					continue
				}
				if !compareCurrentPodWithNewPod(params, pod, node) && pod.DeletionTimestamp == nil {
					result.PodsToDelete = append(result.PodsToDelete, node)
				} else {
					currentPods++
					if podUtils.IsPodAvailable(pod, 0, metaNow) {
						availablePods++
					}
					if podUtils.IsPodReady(pod) {
						readyPods++
					}

					// Check if deploy should be paused due to restarts. Note that pausing the canary will have no effect if it has been validated or failed
					restartCount, reason := podUtils.HighestRestartCount(pod)
					if restartCount == 0 {
						continue
					}

					if autoFailEnabled && !isFailed && restartCount > autoFailMaxRestarts {
						err = failCanaryDeployment(client, daemonset, reason)
						if err != nil {
							params.Logger.Error(err, "Failed to set canary deployment to failed")
						} else {
							params.Logger.V(1).Info("Canary deployment is now failed")
						}
						isFailed = true
					}

					if autoPauseEnabled && !isPaused && !isFailed && restartCount > autoPauseMaxRestarts {
						err = pauseCanaryDeployment(client, daemonset, reason)
						if err != nil {
							params.Logger.Error(err, "Failed to pause canary deployment")
						} else {
							params.Logger.V(1).Info("Canary deployment is now paused")
						}
						isPaused = true
					}

					podRestartTime := podUtils.MostRecentRestartTime(pod)
					if podRestartTime.After(newRestartTime) {
						newRestartTime = podRestartTime
						restartingPod = pod.ObjectMeta.Name
					}
				}
			}
		}
	}
	result.NewStatus = params.NewStatus.DeepCopy()
	result.NewStatus.Status = string(ReplicaSetStatusCanary)
	result.NewStatus.Desired = desiredPods
	result.NewStatus.Ready = readyPods
	result.NewStatus.Available = availablePods
	result.NewStatus.Current = currentPods

	if !newRestartTime.IsZero() && newRestartTime.After(lastRestartTime) {
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(
			result.NewStatus,
			metav1.NewTime(newRestartTime),
			v1alpha1.ConditionTypePodRestarting,
			v1.ConditionTrue,
			fmt.Sprintf("Pod %s had a container restart", restartingPod),
			false,
			true,
		)
	}

	restartCondition = conditions.GetExtendedDaemonSetReplicaSetStatusCondition(result.NewStatus, v1alpha1.ConditionTypePodRestarting)
	autoFailMaxRestartsDuration := params.Strategy.Canary.AutoFail.MaxRestartsDuration.Duration
	if !isFailed && restartCondition != nil && restartCondition.LastUpdateTime.Sub(restartCondition.LastTransitionTime.Time) > autoFailMaxRestartsDuration {
		err = failCanaryDeployment(client, daemonset, v1alpha1.ExtendedDaemonSetStatusRestartsTimeoutExceeded)
		if err != nil {
			params.Logger.Error(err, "Failed to set canary deployment to failed")
		} else {
			params.Logger.V(1).Info("Canary deployment is now failed")
		}
	}

	params.Logger.V(1).Info("NewStatus", "Desired", desiredPods, "Ready", readyPods, "Available", availablePods)
	params.Logger.V(1).Info("Result", "PodsToCreate", result.PodsToCreate, "PodsToDelete", result.PodsToDelete)

	// Populate list of unscheduled pods on nodes due to resource limitation
	result.UnscheduledNodesDueToResourcesConstraints = manageUnscheduledPodNodes(params.UnscheduledPods)

	// Cleanup Pods
	result.NewStatus, result.Result, err = cleanupPods(client, params.Logger, result.NewStatus, params.PodToCleanUp)
	if result.NewStatus.Desired != result.NewStatus.Ready || needRequeue {
		result.Result.Requeue = true
		result.Result.RequeueAfter = time.Second
	}

	return result, err
}
