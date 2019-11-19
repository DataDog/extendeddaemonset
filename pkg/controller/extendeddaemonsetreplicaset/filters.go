// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

import (
	"sort"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/controller/extendeddaemonsetreplicaset/scheduler"
	podutils "github.com/datadog/extendeddaemonset/pkg/controller/utils/pod"
)

// FilterAndMapPodsByNode used to map pods by associated node. It also return the list of pods that
// should be deleted (not needed anymore), and pods that are not scheduled yet (created but not scheduled)
func FilterAndMapPodsByNode(logger logr.Logger, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet,
	nodeList *corev1.NodeList, podList *corev1.PodList, ignoreNodes []string) (podByNode map[string]*corev1.Pod,
	podToDelete, unscheduledPods []*corev1.Pod) {
	// For faster search convert slice to map
	ignoreMapNode := make(map[string]bool)
	for _, name := range ignoreNodes {
		ignoreMapNode[name] = true
	}

	// create a Fake pod from the current replicaset.spec.template
	newPod := podutils.CreatePodFromDaemonSetReplicaSet(nil, replicaset, "", false)
	// var unschedulabledNodes []*corev1.Node
	// Associate Pods to Nodes
	podsByNodeName := make(map[string][]*corev1.Pod)
	for _, node := range nodeList.Items {
		if _, ok := ignoreMapNode[node.Name]; ok {
			continue
		}
		// Filter Nodes Unschedulabled
		if scheduler.CheckNodeFitness(logger.WithValues("filter", "FilterAndMapPodsByNode"), newPod, &node, false) {
			podsByNodeName[node.Name] = nil
		} else {
			logger.V(1).Info("CheckNodeFitness not ok", "reason", "DeletionTimestamp==nil", "node.Name", node.Name)
		}
	}

	for id, pod := range podList.Items {
		if _, scheduled := podutils.IsPodScheduled(&pod); !scheduled {
			unscheduledPods = append(unscheduledPods, &podList.Items[id])
			continue
		}
		if _, ok := podsByNodeName[pod.Spec.NodeName]; ok {
			// ignore pod with status phase unknown: usually it means the pod's node
			// in unreacheable so the pod can't be delete. It will be cleanup by the
			// pods garbage collector.
			if pod.Status.Phase == corev1.PodUnknown {
				continue
			}

			podsByNodeName[pod.Spec.NodeName] = append(podsByNodeName[pod.Spec.NodeName], &podList.Items[id])
		} else {
			if _, ok := ignoreMapNode[pod.Spec.NodeName]; ok {
				continue
			}
			// ignore pod with status phase unknown: usually it means the pod's node
			// in unreacheable so the pod can't be delete. It will be cleanup by the
			// pods garbage collector.
			if pod.Status.Phase == corev1.PodUnknown {
				continue
			}
			// Add pod with missing Node in podToDelete slice
			// Skip pod with DeletionTimestamp already set
			if pod.DeletionTimestamp == nil {
				podToDelete = append(podToDelete, &podList.Items[id])
				logger.V(1).Info("PodToDelete", "reason", "DeletionTimestamp==nil", "pod.Name", pod.Name, "node.Name", pod.Spec.NodeName)
			}
		}
	}

	// filter pod node, remove duplicated
	var duplicatedPods []*corev1.Pod
	podByNode, duplicatedPods = FilterPodsByNode(podsByNodeName)

	// add duplicated pods to the pod deletion slice
	for _, pod := range duplicatedPods {
		logger.V(1).Info("PodToDelete", "reason", "duplicatedPod", "pod.Name", pod.Name, "node.Name", pod.Spec.NodeName)
	}
	podToDelete = append(podToDelete, duplicatedPods...)

	// Filter Pods in Terminated state
	return podByNode, podToDelete, unscheduledPods
}

// FilterPodsByNode if several Pods are listed for the same Node select "best" Pod one, and add other pod to
// the deletion pod slice
func FilterPodsByNode(podsByNodeName map[string][]*corev1.Pod) (map[string]*corev1.Pod, []*corev1.Pod) {
	// filter pod node, remove duplicated
	podByNodeName := map[string]*corev1.Pod{}
	duplicatedPods := []*corev1.Pod{}
	for nodeName, pods := range podsByNodeName {
		podByNodeName[nodeName] = nil
		sort.Sort(podByCreationTimestampAndPhase(pods))
		for id := range pods {
			if id == 0 {
				podByNodeName[nodeName] = pods[id]
			} else {
				duplicatedPods = append(duplicatedPods, pods[id])
			}
		}
	}

	return podByNodeName, duplicatedPods
}

type podByCreationTimestampAndPhase []*corev1.Pod

func (o podByCreationTimestampAndPhase) Len() int      { return len(o) }
func (o podByCreationTimestampAndPhase) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o podByCreationTimestampAndPhase) Less(i, j int) bool {
	// Scheduled Pod first
	if len(o[i].Spec.NodeName) != 0 && len(o[j].Spec.NodeName) == 0 {
		return true
	}

	if len(o[i].Spec.NodeName) == 0 && len(o[j].Spec.NodeName) != 0 {
		return false
	}

	if o[i].CreationTimestamp.Equal(&o[j].CreationTimestamp) {
		return o[i].Name < o[j].Name
	}
	return o[i].CreationTimestamp.Before(&o[j].CreationTimestamp)
}
