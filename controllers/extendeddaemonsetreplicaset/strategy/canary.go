// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
	eds "github.com/DataDog/extendeddaemonset/controllers/extendeddaemonset"
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
	maxRestarts := int(*daemonset.Spec.Strategy.Canary.AutoPause.MaxRestarts)

	// Canary mode
	for _, nodeName := range params.CanaryNodes {
		node := params.NodeByName[nodeName]
		desiredPods++
		if pod, ok := params.PodByNodeName[node]; ok {
			if pod == nil {
				result.PodsToCreate = append(result.PodsToCreate, node)
			} else {
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
					if autoPauseEnabled && !isPaused {
						// Check if deploy should be paused due to restarts. Note that pausing the canary will have no effect if it has been validated or failed
						if isRestarting, reason := podUtils.IsPodRestarting(pod, maxRestarts); isRestarting {
							err = pauseCanaryDeployment(client, daemonset, reason)
							if err != nil {
								params.Logger.Error(err, "Failed to pause canary deployment")
							} else {
								params.Logger.V(1).Info("Canary deployment paused")
							}
							isPaused = true
						}
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
