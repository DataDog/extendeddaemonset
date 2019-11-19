// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/controller/extendeddaemonsetreplicaset/conditions"
	"github.com/datadog/extendeddaemonset/pkg/controller/extendeddaemonsetreplicaset/strategy/limits"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils"
	podutils "github.com/datadog/extendeddaemonset/pkg/controller/utils/pod"
)

// ManageDeployment used to manage ReplicaSet in rollingupdate state
func ManageDeployment(client client.Client, params *Parameters) (*Result, error) {
	result := &Result{}

	// remove canary node if define
	for _, nodeName := range params.CanaryNodes {
		delete(params.PodByNodeName, nodeName)
	}
	now := time.Now()
	metaNow := metav1.NewTime(now)
	var desiredPods, availablePods, readyPods, currentPods, oldAvailablePods, podsTerminating int32

	allPodToCreate := []string{}
	allPodToDelete := []string{}

	for nodeName, pod := range params.PodByNodeName {
		desiredPods++
		if pod == nil {
			allPodToCreate = append(allPodToCreate, nodeName)
		} else {
			if !compareSpecTemplateMD5Hash(params.Replicaset.Spec.TemplateGeneration, pod) {
				if pod.DeletionTimestamp == nil {
					allPodToDelete = append(allPodToDelete, nodeName)
				} else {
					podsTerminating++
					continue
				}
				if podutils.IsPodAvailable(pod, 0, metaNow) {
					oldAvailablePods++
				}
			} else {
				currentPods++
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
	nbNodes := len(params.PodByNodeName)

	maxUnavailable, err := intstrutil.GetValueFromIntOrPercent(params.Replicaset.Spec.Strategy.RollingUpdate.MaxUnavailable, nbNodes, true)
	if err != nil {
		params.Logger.Error(err, "unable to retrieve maxUnavailable pod from the strategy.RollingUpdate.MaxUnavailable parameter")
		return result, err
	}
	params.Logger.V(1).Info("Parameters", "nbNodes", nbNodes, "createdPod", currentPods, "nbPodReady", readyPods, "availablePods", availablePods, "oldAvailablePods", oldAvailablePods, "maxUnavailable", maxUnavailable, "nbPodToCreate", len(allPodToCreate), "nbPodToDelete", len(allPodToDelete), "podsTerminating", podsTerminating)

	rollingUpdateStartTime := getRollingUpdateStartTime(&params.Replicaset.Status, now)
	maxCreation, err := calculateMaxCreation(&params.Replicaset.Spec.Strategy.RollingUpdate, nbNodes, rollingUpdateStartTime, now)
	if err != nil {
		params.Logger.Error(err, "error during calculateMaxCreation execution")
		return result, err
	}

	limitParams := limits.Parameters{
		NbNodes: nbNodes,

		NbPods:             int(currentPods),
		NbAvailablesPod:    int(availablePods),
		NbOldAvailablesPod: int(oldAvailablePods),
		NbCreatedPod:       int(currentPods),
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
		result.NewStatus.Current = currentPods
		result.NewStatus.Available = availablePods
	}

	// Populate list of unscheduled pods on nodes due to resource limitation
	result.UnscheduledNodesDueToResourcesConstraints = manageUnscheduledPodNodes(params.UnscheduledPods)
	// Cleanup Pods
	result.NewStatus, result.Result, err = cleanupPods(client, params.Logger, result.NewStatus, params.PodToCleanUp)
	if result.NewStatus.Desired != result.NewStatus.Ready {
		result.Result.Requeue = true
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
