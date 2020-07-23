// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonset

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/labels"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
)

// IsCanaryPhaseEnded used to know if the Canary duration has finished.
// If the duration is completed: return true
// If the duration is not completed: return false and the remaining duration.
func IsCanaryDeploymentEnded(specCanary *datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary, rs *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, now time.Time) (bool, time.Duration) {
	var pendingDuration time.Duration
	if specCanary == nil {
		return true, pendingDuration
	}
	// if specCanary.Paused || specCanary.Duration == nil {
	if specCanary.Duration == nil {
		// in this case, it means the canary never ends
		return false, pendingDuration
	}
	pendingDuration = rs.CreationTimestamp.Add(specCanary.Duration.Duration).Sub(now)
	if rs.CreationTimestamp.Add(specCanary.Duration.Duration).Sub(now) >= 0 {
		return false, pendingDuration
	}

	return true, pendingDuration
}

// IsCanaryPhasePaused checks if the Canary deployment has been paused
func IsCanaryDeploymentPaused(dsAnnotations map[string]string) (bool, datadoghqv1alpha1.ExtendedDaemonSetStatusReason) {
	isPaused, found := dsAnnotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey]
	if found && isPaused == "true" {
		if reason, found := dsAnnotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedReasonAnnotationKey]; found {
			return true, datadoghqv1alpha1.ExtendedDaemonSetStatusReason(reason)
		}
		return true, datadoghqv1alpha1.ExtendedDaemonSetStatusReasonUnknown
	}
	return false, ""
}

// IsCanaryDeploymentValid used to know if the Canary deployment has been declared
// valid even if its duration has not finished yet.
// If the ExtendedDaemonSet has the corresponding annotation: return true
func IsCanaryDeploymentValid(dsAnnotations map[string]string, rsName string) bool {
	if value, found := dsAnnotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryValidAnnotationKey]; found {
		return value == rsName
	}
	return false
}

func getPodListFromReplicaSet(c client.Client, ds *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	podSelector := labels.Set{datadoghqv1alpha1.ExtendedDaemonSetReplicaSetNameLabelKey: ds.Name}
	podListOptions := []client.ListOption{
		&client.MatchingLabelsSelector{Selector: podSelector.AsSelectorPreValidated()},
	}
	if err := c.List(context.TODO(), podList, podListOptions...); err != nil {
		return nil, err
	}
	return podList, nil
}
