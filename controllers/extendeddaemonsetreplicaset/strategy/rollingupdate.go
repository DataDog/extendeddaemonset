// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/conditions"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/strategy/limits"
	"github.com/DataDog/extendeddaemonset/pkg/controller/utils"
	podutils "github.com/DataDog/extendeddaemonset/pkg/controller/utils/pod"
)

// cleanCanaryLabelsThreshold is the duration since the last transition to a rolling update of a replicaset
// during which we keep retrying cleaning up the canary labels that were added to the canary pods during the canary phase.
const cleanCanaryLabelsThreshold = 5 * time.Minute

// ManageDeployment used to manage ReplicaSet in rollingupdate state.
func ManageDeployment(client runtimeclient.Client, params *Parameters) (*Result, error) {
	result := &Result{}

	// remove canary node if define.
	for _, nodeName := range params.CanaryNodes {
		delete(params.PodByNodeName, params.NodeByName[nodeName])
	}
	now := time.Now()
	metaNow := metav1.NewTime(now)
	var desiredPods, availablePods, readyPods, createdPods, allPods, oldAvailablePods, podsTerminating, nbIgnoredUnresponsiveNodes int32

	allPodToCreate := []*NodeItem{}
	allPodToDelete := []*NodeItem{}

	nbNodes := len(params.PodByNodeName)

	maxPodSchedulerFailure, err := intstrutil.GetValueFromIntOrPercent(params.Strategy.RollingUpdate.MaxPodSchedulerFailure, nbNodes, true)
	if err != nil {
		params.Logger.Error(err, "unable to retrieve maxPodSchedulerFailure from the strategy.RollingUpdate.MaxPodSchedulerFailure parameter")

		return result, err
	}

	for node, pod := range params.PodByNodeName {
		desiredPods++
		if pod == nil {
			allPodToCreate = append(allPodToCreate, node)
		} else {
			if podutils.HasPodSchedulerIssue(pod) && int(nbIgnoredUnresponsiveNodes) < maxPodSchedulerFailure {
				nbIgnoredUnresponsiveNodes++

				continue
			}

			allPods++
			if !compareCurrentPodWithNewPod(params, pod, node) {
				if pod.DeletionTimestamp == nil {
					allPodToDelete = append(allPodToDelete, node)
				} else {
					podsTerminating++

					continue
				}
				if podutils.IsPodAvailable(pod, 0, metaNow) {
					oldAvailablePods++
				}
			} else {
				createdPods++
				if podutils.IsPodAvailable(pod, 0, metaNow) {
					availablePods++
				}
				if podutils.IsPodReady(pod) {
					readyPods++
				}
			}
		}
	}

	// Retrieves parameters for calculation
	maxUnavailable, err := intstrutil.GetValueFromIntOrPercent(params.Strategy.RollingUpdate.MaxUnavailable, nbNodes, true)
	if err != nil {
		params.Logger.Error(err, "unable to retrieve maxUnavailable pod from the strategy.RollingUpdate.MaxUnavailable parameter")

		return result, err
	}

	rollingUpdateStartTime := getRollingUpdateStartTime(&params.Replicaset.Status, now)
	maxCreation, err := calculateMaxCreation(&params.Strategy.RollingUpdate, nbNodes, rollingUpdateStartTime, now)
	if err != nil {
		params.Logger.Error(err, "error during calculateMaxCreation execution")

		return result, err
	}
	params.Logger.V(1).Info("Parameters", "nbNodes", nbNodes, "createdPods", createdPods, "allPods", allPods, "nbPodReady", readyPods, "availablePods", availablePods, "oldAvailablePods", oldAvailablePods, "maxPodsCreation", maxCreation, "maxUnavailable", maxUnavailable, "nbPodToCreate", len(allPodToCreate), "nbPodToDelete", len(allPodToDelete), "podsTerminating", podsTerminating)

	limitParams := limits.Parameters{
		NbNodes: nbNodes,

		NbPods:             int(allPods),
		NbAvailablesPod:    int(availablePods),
		NbOldAvailablesPod: int(oldAvailablePods),
		NbCreatedPod:       int(createdPods),
		MaxUnavailablePod:  maxUnavailable,
		MaxPodCreation:     maxCreation,
	}
	nbPodToCreate, nbPodToDelete := limits.CalculatePodToCreateAndDelete(limitParams)
	nbPodToDeleteWithConstraint := utils.MinInt(nbPodToDelete, len(allPodToDelete))
	nbPodToCreateWithConstraint := utils.MinInt(nbPodToCreate, len(allPodToCreate))
	params.Logger.V(1).Info("Pods actions with limits", "nbPodToDelete", nbPodToDelete, "nbPodToCreate", nbPodToCreate, "nbPodToDeleteWithConstraint", nbPodToDeleteWithConstraint, "nbPodToCreateWithConstraint", nbPodToCreateWithConstraint)

	result.PodsToDelete = allPodToDelete[:nbPodToDeleteWithConstraint]
	result.PodsToCreate = allPodToCreate[:nbPodToCreateWithConstraint]
	{
		result.NewStatus = params.NewStatus.DeepCopy()
		result.NewStatus.Status = string(ReplicaSetStatusActive)
		result.NewStatus.Desired = desiredPods
		result.NewStatus.Ready = readyPods
		result.NewStatus.Current = createdPods
		result.NewStatus.Available = availablePods
		result.NewStatus.IgnoredUnresponsiveNodes = nbIgnoredUnresponsiveNodes
	}

	// Populate list of unscheduled pods on nodes due to resource limitation
	result.UnscheduledNodesDueToResourcesConstraints = manageUnscheduledPodNodes(params.UnscheduledPods)
	// Cleanup Pods
	err = cleanupPods(client, params.Logger, result.NewStatus, params.PodToCleanUp)
	if result.NewStatus.Desired != result.NewStatus.Ready {
		result.Result.Requeue = true
	}

	// Remove canary labels from canary pods (if they exist)
	// We keep retrying these operations only for the first X minutes after starting the rolling update to avoid Listing pods endlessly.
	if time.Since(rollingUpdateStartTime) < cleanCanaryLabelsThreshold {
		canaryPods := &corev1.PodList{}
		listOptions := []runtimeclient.ListOption{
			runtimeclient.MatchingLabels{
				datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelValue,
				datadoghqv1alpha1.ExtendedDaemonSetReplicaSetNameLabelKey:   params.Replicaset.GetName(),
			},
		}
		if err = client.List(context.TODO(), canaryPods, listOptions...); err != nil {
			params.Logger.Error(err, "Couldn't get canary pods")
			result.Result.Requeue = true
		} else {
			for _, pod := range canaryPods.Items {
				if err = deletePodLabel(params.Logger, client, &pod, datadoghqv1alpha1.ExtendedDaemonSetReplicaSetCanaryLabelKey); err != nil {
					params.Logger.Error(err, fmt.Sprintf("Couldn't remove canary label from pod '%s/%s'", pod.GetNamespace(), pod.GetName()))
					result.Result.Requeue = true
				}
			}
		}
	}

	return result, err
}

func getRollingUpdateStartTime(status *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus, now time.Time) time.Time {
	if status == nil {
		return now
	}
	cond := conditions.GetExtendedDaemonSetReplicaSetStatusCondition(status, datadoghqv1alpha1.ConditionTypeActive)
	if cond == nil {
		return now
	}
	if cond.Status == corev1.ConditionTrue {
		return cond.LastTransitionTime.Time
	}

	return now
}

func calculateMaxCreation(params *datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate, nbNodes int, rsStartTime, now time.Time) (int, error) {
	startValue, err := intstrutil.GetValueFromIntOrPercent(params.SlowStartAdditiveIncrease, nbNodes, true)
	if err != nil {
		return 0, err
	}
	rollingUpdateDuration := now.Sub(rsStartTime)
	nbSlowStartSlot := int(rollingUpdateDuration / params.SlowStartIntervalDuration.Duration)
	result := (1 + nbSlowStartSlot) * startValue
	if result > int(*params.MaxParallelPodCreation) {
		result = int(*params.MaxParallelPodCreation)
	}

	return result, nil
}
