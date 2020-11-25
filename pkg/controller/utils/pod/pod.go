// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package pod

import (
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/pkg/controller/utils/affinity"
)

// GetContainerStatus extracts the status of container "name" from "statuses".
// It also returns if "name" exists.
func GetContainerStatus(statuses []v1.ContainerStatus, name string) (v1.ContainerStatus, bool) {
	for i := range statuses {
		if statuses[i].Name == name {
			return statuses[i], true
		}
	}
	return v1.ContainerStatus{}, false
}

// GetExistingContainerStatus extracts the status of container "name" from "statuses",
func GetExistingContainerStatus(statuses []v1.ContainerStatus, name string) v1.ContainerStatus {
	status, _ := GetContainerStatus(statuses, name)
	return status
}

// IsPodScheduled return true if it is already assigned to a Node
func IsPodScheduled(pod *v1.Pod) (string, bool) {
	isScheduled := pod.Spec.NodeName != ""
	nodeName := affinity.GetNodeNameFromAffinity(pod.Spec.Affinity)
	return nodeName, isScheduled
}

// IsPodAvailable returns true if a pod is available; false otherwise.
// Precondition for an available pod is that it must be ready. On top
// of that, there are two cases when a pod can be considered available:
// 1. minReadySeconds == 0, or
// 2. LastTransitionTime (is set) + minReadySeconds < current time
func IsPodAvailable(pod *v1.Pod, minReadySeconds int32, now metav1.Time) bool {
	if IsPodReady(pod) {
		return true
	}

	c := GetPodReadyCondition(pod.Status)
	minReadySecondsDuration := time.Duration(minReadySeconds) * time.Second
	if minReadySeconds == 0 || !c.LastTransitionTime.IsZero() && c.LastTransitionTime.Add(minReadySecondsDuration).Before(now.Time) {
		return true
	}
	return false
}

// IsPodReady returns true if a pod is ready; false otherwise.
func IsPodReady(pod *v1.Pod) bool {
	return IsPodReadyConditionTrue(pod.Status)
}

// IsPodReadyConditionTrue returns true if a pod is ready; false otherwise.
func IsPodReadyConditionTrue(status v1.PodStatus) bool {
	condition := GetPodReadyCondition(status)
	return condition != nil && condition.Status == v1.ConditionTrue
}

// GetPodReadyCondition extracts the pod ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetPodReadyCondition(status v1.PodStatus) *v1.PodCondition {
	_, condition := GetPodCondition(&status, v1.PodReady)
	return condition
}

// GetPodCondition extracts the provided condition from the given status and returns that.
// Returns nil and -1 if the condition is not present, and the index of the located condition.
func GetPodCondition(status *v1.PodStatus, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if status == nil {
		return -1, nil
	}
	return GetPodConditionFromList(status.Conditions, conditionType)
}

// GetPodConditionFromList extracts the provided condition from the given list of condition and
// returns the index of the condition and the condition. Returns -1 and nil if the condition is not present.
func GetPodConditionFromList(conditions []v1.PodCondition, conditionType v1.PodConditionType) (int, *v1.PodCondition) {
	if conditions == nil {
		return -1, nil
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return i, &conditions[i]
		}
	}
	return -1, nil
}

// HighestRestartCount checks if a pod in the Canary deployment is restarting
// This returns the count and the "reason" for the pod with the most restarts
func HighestRestartCount(pod *v1.Pod) (int, datadoghqv1alpha1.ExtendedDaemonSetStatusReason) {
	// track the highest number of restarts among pod containers
	var restartCount int
	var reason datadoghqv1alpha1.ExtendedDaemonSetStatusReason
	for _, s := range pod.Status.ContainerStatuses {
		if restartCount < int(s.RestartCount) {
			restartCount = int(s.RestartCount)
			reason = datadoghqv1alpha1.ExtendedDaemonSetStatusReasonUnknown
			if s.LastTerminationState != (v1.ContainerState{}) && *s.LastTerminationState.Terminated != (v1.ContainerStateTerminated{}) {
				switch s.LastTerminationState.Terminated.Reason {
				case string(datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB):
					reason = datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB
				case string(datadoghqv1alpha1.ExtendedDaemonSetStatusReasonOOM):
					reason = datadoghqv1alpha1.ExtendedDaemonSetStatusReasonOOM
				}
			}
		}
	}
	return restartCount, reason
}

// MostRecentRestartTime returns the most recent restart time for a pod or the time
func MostRecentRestartTime(pod *v1.Pod) time.Time {
	var recentRestartTime time.Time
	for _, s := range pod.Status.ContainerStatuses {
		if s.RestartCount != 0 && s.LastTerminationState != (v1.ContainerState{}) && s.LastTerminationState.Terminated != (&v1.ContainerStateTerminated{}) {
			if s.LastTerminationState.Terminated.FinishedAt.After(recentRestartTime) {
				recentRestartTime = s.LastTerminationState.Terminated.FinishedAt.Time
			}
		}
	}
	return recentRestartTime
}

// HasPodSchedulerIssue returns true if a pod remained unscheduled for more than 10 minutes
// or if it stayed in `Terminating` state for longer than its grace period.
func HasPodSchedulerIssue(pod *v1.Pod) bool {
	_, isScheduled := IsPodScheduled(pod)
	if !isScheduled && pod.CreationTimestamp.Add(10*time.Minute).Before(time.Now()) {
		return true
	}

	if pod.DeletionTimestamp != nil && pod.DeletionGracePeriodSeconds != nil &&
		pod.DeletionTimestamp.Add(time.Duration(*pod.DeletionGracePeriodSeconds)*time.Second).Before(time.Now()) {
		return true
	}

	return false
}

// UpdatePodCondition updates existing pod condition or creates a new one. Sets LastTransitionTime to now if the
// status has changed.
// Returns true if pod condition has changed or has been added.
func UpdatePodCondition(status *v1.PodStatus, condition *v1.PodCondition) bool {
	condition.LastTransitionTime = metav1.Now()
	// Try to find this pod condition.
	conditionIndex, oldCondition := GetPodCondition(status, condition.Type)

	if oldCondition == nil {
		// We are adding new pod condition.
		status.Conditions = append(status.Conditions, *condition)
		return true
	}
	// We are updating an existing condition, so we need to check if it has changed.
	if condition.Status == oldCondition.Status {
		condition.LastTransitionTime = oldCondition.LastTransitionTime
	}

	isEqual := condition.Status == oldCondition.Status &&
		condition.Reason == oldCondition.Reason &&
		condition.Message == oldCondition.Message &&
		condition.LastProbeTime.Equal(&oldCondition.LastProbeTime) &&
		condition.LastTransitionTime.Equal(&oldCondition.LastTransitionTime)

	status.Conditions[conditionIndex] = *condition
	// Return true if one of the fields have changed.
	return !isEqual
}

// IsEvicted returns whether the status corresponds to an evicted pod
func IsEvicted(status *v1.PodStatus) bool {
	if status.Phase == v1.PodFailed && status.Reason == "Evicted" {
		return true
	}
	return false
}

// SortPodByCreationTime return the pods sorted by creation time
// from the newer to the older
func SortPodByCreationTime(pods []*v1.Pod) []*v1.Pod {
	sort.Sort(podByCreationTimestamp(pods))
	return pods
}

type podByCreationTimestamp []*v1.Pod

func (o podByCreationTimestamp) Len() int      { return len(o) }
func (o podByCreationTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o podByCreationTimestamp) Less(i, j int) bool {
	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name > o[j].Name
	}
	return o[j].CreationTimestamp.Before(&o[i].CreationTimestamp)
}
