// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package strategy

import (
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
	eds "github.com/DataDog/extendeddaemonset/controllers/extendeddaemonset"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/conditions"
	podUtils "github.com/DataDog/extendeddaemonset/pkg/controller/utils/pod"
)

// ManageCanaryDeployment used to manage ReplicaSet in Canary state
func ManageCanaryDeployment(client client.Client, daemonset *v1alpha1.ExtendedDaemonSet, params *Parameters) (*Result, error) {
	// Manage canary status
	result := manageCanaryStatus(daemonset.GetAnnotations(), params)

	if result.IsFailed {
		err := failCanaryDeployment(client, daemonset, result.FailedReason)
		if err != nil {
			params.Logger.Error(err, "Failed to set canary deployment to failed")
			result.Result = requeuePromptly()
		} else {
			params.Logger.V(1).Info("Canary deployment is now failed")
		}
	} else if result.IsPaused {
		err := pauseCanaryDeployment(client, daemonset, result.PausedReason)
		if err != nil {
			params.Logger.Error(err, "Failed to pause canary deployment")
			result.Result = requeuePromptly()
		} else {
			params.Logger.V(1).Info("Canary deployment is now paused")
		}
	}

	err := ensureCanaryPodLabels(client, params)
	if err != nil {
		result.Result = requeuePromptly()
	}

	// Populate list of unscheduled pods on nodes due to resource limitation
	result.UnscheduledNodesDueToResourcesConstraints = manageUnscheduledPodNodes(params.UnscheduledPods)

	// Cleanup Pods
	err = cleanupPods(client, params.Logger, result.NewStatus, params.PodToCleanUp)
	if err != nil {
		result.Result = requeuePromptly()
	}

	return result, nil
}

// manageCanaryStatus manages ReplicaSet status in Canary state
func manageCanaryStatus(annotations map[string]string, params *Parameters) *Result {
	result := &Result{}
	result.NewStatus = params.NewStatus.DeepCopy()
	result.NewStatus.Status = string(ReplicaSetStatusCanary)

	result.IsFailed = eds.IsCanaryDeploymentFailed(annotations)
	result.IsPaused, _ = eds.IsCanaryDeploymentPaused(annotations)

	var (
		now     = time.Now()
		metaNow = metav1.NewTime(now)

		desiredPods, currentPods, availablePods, readyPods int32

		needRequeue            bool
		podsToCheckForRestarts []*v1.Pod

		podsToCreate []*NodeItem
		podsToDelete []*NodeItem
	)

	// First scan canary node list for pods to create or delete
	for _, nodeName := range params.CanaryNodes {
		node := params.NodeByName[nodeName]
		desiredPods++
		if pod, ok := params.PodByNodeName[node]; ok {
			if pod == nil {
				podsToCreate = append(podsToCreate, node)
				continue
			}

			if pod.DeletionTimestamp != nil {
				needRequeue = true
				continue
			}

			if !compareCurrentPodWithNewPod(params, pod, node) {
				podsToDelete = append(podsToDelete, node)
				continue
			}

			currentPods++
			if podUtils.IsPodAvailable(pod, 0, metaNow) {
				availablePods++
			}
			if podUtils.IsPodReady(pod) {
				readyPods++
			}

			podsToCheckForRestarts = append(podsToCheckForRestarts, pod)
		}
	}

	// Update result to reflect active pods currently experiencing restarts
	// potentially placing canary into paused or failed state
	manageCanaryPodRestarts(podsToCheckForRestarts, params, result)

	// Update pod counts
	result.NewStatus.Desired = desiredPods
	result.NewStatus.Ready = readyPods
	result.NewStatus.Available = availablePods
	result.NewStatus.Current = currentPods

	result.PodsToDelete = podsToDelete

	// Do not create any pods if canary is paused or failed
	if len(podsToCreate) != 0 && !result.IsPaused && !result.IsFailed {
		result.PodsToCreate = podsToCreate
		needRequeue = true
	}

	params.Logger.V(1).Info("NewStatus", "Desired", desiredPods, "Ready", readyPods, "Available", availablePods)
	params.Logger.V(1).Info(
		"Result",
		"PodsToCreate", result.PodsToCreate,
		"PodsToDelete", result.PodsToDelete,
		"IsFailed", result.IsFailed,
		"FailedReason", result.FailedReason,
		"IsPaused", result.IsPaused,
		"PausedReason", result.PausedReason,
	)
	params.Logger.V(1).Info("IsFailed", "PodsToCreate", result.PodsToCreate, "PodsToDelete", result.PodsToDelete)

	if needRequeue || !result.IsFailed && !result.IsPaused && result.NewStatus.Desired != result.NewStatus.Ready {
		result.Result = requeuePromptly()
	}
	return result
}

// manageCanaryPodRestarts checks if canary should be failed or paused due to restarts.
// Note that pausing the canary will have no effect if it has been validated or failed
func manageCanaryPodRestarts(pods []*v1.Pod, params *Parameters, result *Result) {
	var (
		canary               = params.Strategy.Canary
		autoPauseEnabled     = *canary.AutoPause.Enabled
		autoPauseMaxRestarts = int(*canary.AutoPause.MaxRestarts)

		autoFailEnabled     = *canary.AutoFail.Enabled
		autoFailMaxRestarts = int(*canary.AutoFail.MaxRestarts)

		newRestartTime      time.Time
		restartingPodStatus string
	)

	// Note that we still need to evaluate restarts regardless of the enabled autoPause or autoFail
	// since we maintain the restarting condition that can be checked by canary.noRestartsDuration
	for _, pod := range pods {
		restartCount, highRestartReason := podUtils.HighestRestartCount(pod)
		if restartCount == 0 {
			continue
		}

		restartTime, recentRestartReason := podUtils.MostRecentRestart(pod)
		if restartTime.After(newRestartTime) {
			newRestartTime = restartTime
			restartingPodStatus = fmt.Sprintf("Pod %s restarting with reason: %s", pod.ObjectMeta.Name, string(recentRestartReason))
		}

		if result.IsFailed {
			continue
		}

		if autoFailEnabled && restartCount > autoFailMaxRestarts {
			result.IsFailed = true
			result.FailedReason = highRestartReason
			params.Logger.Info(
				"AutoFailed",
				"RestartCount", restartCount,
				"MaxRestarts", autoFailMaxRestarts,
				"Reason", highRestartReason,
			)
			continue
		}

		if !result.IsPaused && autoPauseEnabled && restartCount > autoPauseMaxRestarts {
			result.IsPaused = true
			result.PausedReason = highRestartReason
			params.Logger.Info(
				"AutoPaused",
				"RestartCount", restartCount,
				"MaxRestarts", autoFailMaxRestarts,
				"Reason", highRestartReason,
			)
		}
	}

	var lastRestartTime time.Time
	restartCondition := conditions.GetExtendedDaemonSetReplicaSetStatusCondition(params.NewStatus, v1alpha1.ConditionTypePodRestarting)
	if restartCondition != nil {
		lastRestartTime = restartCondition.LastUpdateTime.Time
	}

	// Track pod restart condition in the status
	if !newRestartTime.IsZero() && newRestartTime.After(lastRestartTime) {
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(
			result.NewStatus,
			metav1.NewTime(newRestartTime),
			v1alpha1.ConditionTypePodRestarting,
			v1.ConditionTrue,
			restartingPodStatus,
			false,
			true,
		)
	}

	if !result.IsFailed && autoFailEnabled && canary.AutoFail.MaxRestartsDuration != nil {
		maxRestartsDuration := canary.AutoFail.MaxRestartsDuration.Duration
		// Check if we are exceeding autoFail.maxRestartsDuration and need to auto-fail
		restartCondition = conditions.GetExtendedDaemonSetReplicaSetStatusCondition(result.NewStatus, v1alpha1.ConditionTypePodRestarting)
		if restartCondition != nil && restartCondition.LastUpdateTime.Sub(restartCondition.LastTransitionTime.Time) > maxRestartsDuration {
			result.IsFailed = true
			result.FailedReason = v1alpha1.ExtendedDaemonSetStatusRestartsTimeoutExceeded
		}
	}

	if result.IsFailed {
		result.NewStatus.Status = string(ReplicaSetStatusCanaryFailed)
	}
}

// ensureCanaryPodLabels ensures that canary label is set on canary pods
func ensureCanaryPodLabels(client client.Client, params *Parameters) error {
	for _, nodeName := range params.CanaryNodes {
		node := params.NodeByName[nodeName]
		if pod, ok := params.PodByNodeName[node]; ok && pod != nil {
			err := addPodLabel(
				client,
				pod,
				v1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey,
				v1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelValue,
			)

			if err != nil {
				params.Logger.Error(err, fmt.Sprintf("Couldn't add the canary label for pod '%s/%s', will retry later", pod.GetNamespace(), pod.GetName()))
				return err
			}
		}
	}
	return nil
}

func requeueIn(requeueAfter time.Duration) reconcile.Result {
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: requeueAfter,
	}
}

func requeuePromptly() reconcile.Result {
	return requeueIn(time.Second)
}
