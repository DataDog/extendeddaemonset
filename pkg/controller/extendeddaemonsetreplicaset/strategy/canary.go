// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	podUtils "github.com/datadog/extendeddaemonset/pkg/controller/utils/pod"
)

// ManageCanaryDeployment used to manage ReplicaSet in Canary state
func ManageCanaryDeployment(client client.Client, params *Parameters) (*Result, error) {
	result := &Result{}

	now := time.Now()
	metaNow := metav1.NewTime(now)
	var desiredPods, currentPods, availablePods, readyPods int32

	var needRequeue bool
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
	var err error

	// Cleanup Pods
	result.NewStatus, result.Result, err = cleanupPods(client, params.Logger, result.NewStatus, params.PodToCleanUp)
	if result.NewStatus.Desired != result.NewStatus.Ready || needRequeue {
		result.Result.Requeue = true
		result.Result.RequeueAfter = time.Second
	}

	return result, err
}
