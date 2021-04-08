// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

import (
	"sort"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/scheduler"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/strategy"
	podutils "github.com/DataDog/extendeddaemonset/pkg/controller/utils/pod"
)

var ignoreEvictedPods = false

func init() {
	pflag.BoolVarP(&ignoreEvictedPods, "ignoreEvictedPods", "i", ignoreEvictedPods, "Enabling this will force new pods creation on nodes where pods are evicted")
}

// FilterAndMapPodsByNode used to map pods by associated node. It also return the list of pods that
// should be deleted (not needed anymore), and pods that are not scheduled yet (created but not scheduled).
func FilterAndMapPodsByNode(logger logr.Logger, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet,
	nodeList *strategy.NodeList, podList *corev1.PodList, ignoreNodes []string) (nodesByName map[string]*strategy.NodeItem, podByNode map[*strategy.NodeItem]*corev1.Pod,
	podToDelete, unscheduledPods []*corev1.Pod) {
	// For faster search convert slice to map
	ignoreMapNode := make(map[string]bool)
	for _, name := range ignoreNodes {
		ignoreMapNode[name] = true
	}

	// create a Fake pod from the current replicaset.spec.template
	newPod, _ := podutils.CreatePodFromDaemonSetReplicaSet(nil, replicaset, nil, nil, false)
	// var unschedulabledNodes []*corev1.Node
	// Associate Pods to Nodes
	podsByNodeName := make(map[string][]*corev1.Pod)
	nodesByName = make(map[string]*strategy.NodeItem)
	for id := range nodeList.Items {
		nodeItem := nodeList.Items[id]
		nodesByName[nodeItem.Node.Name] = nodeItem
		if _, ok := ignoreMapNode[nodeItem.Node.Name]; ok {
			continue
		}
		// Filter Nodes Unschedulabled
		if scheduler.CheckNodeFitness(logger.WithValues("filter", "FilterAndMapPodsByNode"), newPod, nodeItem.Node) {
			podsByNodeName[nodeItem.Node.Name] = nil
		} else {
			logger.V(1).Info("CheckNodeFitness not ok", "reason", "DeletionTimestamp==nil", "node.Name", nodeItem.Node.Name)
		}
	}

	for id, pod := range podList.Items {
		nodeName, err := podutils.GetNodeNameFromPod(&pod)
		if err != nil {
			continue
		}
		if _, ok := podsByNodeName[nodeName]; ok {
			// ignore pod with status phase unknown: usually it means the pod's node
			// in unreacheable so the pod can't be delete. It will be cleanup by the
			// pods garbage collector.
			// Ignore evicted pods to try scheduling new pods.
			// Evicted pods will be cleaned up by pods garbage collector.
			if shouldIgnorePod(pod.Status) {
				continue
			}

			podsByNodeName[nodeName] = append(podsByNodeName[nodeName], &podList.Items[id])

			if _, scheduled := podutils.IsPodScheduled(&pod); !scheduled {
				unscheduledPods = append(unscheduledPods, &podList.Items[id])
			}
		} else {
			if _, ok := ignoreMapNode[nodeName]; ok {
				continue
			}

			// ignore pod with status phase unknown: usually it means the pod's node
			// in unreacheable so the pod can't be delete. It will be cleanup by the
			// pods garbage collector.
			// Ignore evicted pods to try scheduling new pods.
			// Evicted pods will be cleaned up by pods garbage collector.
			if shouldIgnorePod(pod.Status) {
				continue
			}

			// Add pod with missing Node in podToDelete slice
			// Skip pod with DeletionTimestamp already set
			if pod.DeletionTimestamp == nil {
				podToDelete = append(podToDelete, &podList.Items[id])
				logger.V(1).Info("PodToDelete", "reason", "DeletionTimestamp==nil", "pod.Name", pod.Name, "node.Name", nodeName)
			}
		}
	}

	// filter pod node, remove duplicated
	var duplicatedPods []*corev1.Pod
	podByNode, duplicatedPods = FilterPodsByNode(podsByNodeName, nodesByName)

	// add duplicated pods to the pod deletion slice
	for _, pod := range duplicatedPods {
		nodeName, _ := podutils.GetNodeNameFromPod(pod)
		logger.V(1).Info("PodToDelete", "reason", "duplicatedPod", "pod.Name", pod.Name, "node.Name", nodeName)
	}
	podToDelete = append(podToDelete, duplicatedPods...)

	// Filter Pods in Terminated state
	return nodesByName, podByNode, podToDelete, unscheduledPods
}

// FilterPodsByNode if several Pods are listed for the same Node select "best" Pod one, and add other pod to
// the deletion pod slice.
func FilterPodsByNode(podsByNodeName map[string][]*corev1.Pod, nodesMap map[string]*strategy.NodeItem) (map[*strategy.NodeItem]*corev1.Pod, []*corev1.Pod) {
	// filter pod node, remove duplicated
	podByNodeName := map[*strategy.NodeItem]*corev1.Pod{}
	duplicatedPods := []*corev1.Pod{}
	for node, pods := range podsByNodeName {
		podByNodeName[nodesMap[node]] = nil
		sort.Sort(sortPodByNodeName(pods))
		for id := range pods {
			if id == 0 {
				podByNodeName[nodesMap[node]] = pods[id]
			} else {
				duplicatedPods = append(duplicatedPods, pods[id])
			}
		}
	}

	return podByNodeName, duplicatedPods
}

// shouldIgnorePod returns true if the pod is in an unknown phase or was evicted
// if ignoreEvictedPods is disabled, only the unknown phase will be considered.
func shouldIgnorePod(status corev1.PodStatus) bool {
	return status.Phase == corev1.PodUnknown || (ignoreEvictedPods && podutils.IsEvicted(&status))
}

type sortPodByNodeName []*corev1.Pod

func (o sortPodByNodeName) Len() int      { return len(o) }
func (o sortPodByNodeName) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o sortPodByNodeName) Less(i, j int) bool {
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
