// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilserrors "k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/conditions"
	podaffinity "github.com/DataDog/extendeddaemonset/pkg/controller/utils/affinity"
	"github.com/DataDog/extendeddaemonset/pkg/controller/utils/comparison"
	podutils "github.com/DataDog/extendeddaemonset/pkg/controller/utils/pod"
)

const valueTrue = "true"

func compareCurrentPodWithNewPod(params *Parameters, pod *corev1.Pod, node *NodeItem) bool {
	// check that the pod corresponds to the replicaset. if not return false
	if !compareSpecTemplateMD5Hash(params.Replicaset.Spec.TemplateGeneration, pod) {
		return false
	}
	if !compareWithExtendedDaemonsetSettingOverwrite(pod, node) {
		return false
	}
	if !compareNodeResourcesOverwriteMD5Hash(params.EDSName, params.Replicaset, pod, node) {
		return false
	}
	return true
}

func compareNodeResourcesOverwriteMD5Hash(edsName string, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, pod *corev1.Pod, node *NodeItem) bool {
	nodeHash := comparison.GenerateHashFromEDSResourceNodeAnnotation(replicaset.Namespace, edsName, node.Node.GetAnnotations())
	if val, ok := pod.Annotations[datadoghqv1alpha1.MD5NodeExtendedDaemonSetAnnotationKey]; !ok && nodeHash == "" || ok && val == nodeHash {
		return true
	}
	return false
}

func compareWithExtendedDaemonsetSettingOverwrite(pod *corev1.Pod, node *NodeItem) bool {
	if node.ExtendedDaemonsetSetting != nil {
		specCopy := pod.Spec.DeepCopy()
		for id, container := range specCopy.Containers {
			for _, container2 := range node.ExtendedDaemonsetSetting.Spec.Containers {
				if container.Name == container2.Name {
					for key, val := range container2.Resources.Limits {
						specCopy.Containers[id].Resources.Limits[key] = val
					}
					for key, val := range container2.Resources.Requests {
						specCopy.Containers[id].Resources.Requests[key] = val
					}
					break
				}
			}
		}
		if !apiequality.Semantic.DeepEqual(&pod.Spec, specCopy) {
			return false
		}
	}

	return true
}

func compareSpecTemplateMD5Hash(hash string, pod *corev1.Pod) bool {
	if val, ok := pod.Annotations[datadoghqv1alpha1.MD5ExtendedDaemonSetAnnotationKey]; ok && val == hash {
		return true
	}
	return false
}

func cleanupPods(client client.Client, logger logr.Logger, status *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus, pods []*corev1.Pod) error {
	errs := deletePodSlice(client, logger, pods)
	now := metav1.NewTime(time.Now())
	conditionStatus := corev1.ConditionTrue
	if len(errs) > 0 {
		conditionStatus = corev1.ConditionFalse
	}
	if len(pods) != 0 {
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(status, now, datadoghqv1alpha1.ConditionTypePodsCleanupDone, conditionStatus, "", false, false)
	}
	return utilserrors.NewAggregate(errs)
}

func deletePodSlice(client client.Client, logger logr.Logger, podsToDelete []*corev1.Pod) []error {
	var errs []error
	var wg sync.WaitGroup
	for id, pod := range podsToDelete {
		if pod.DeletionTimestamp != nil {
			// already in deletion phase
			continue
		}
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pod := podsToDelete[id]
			logger.Info("cleanupPods delete pod", "pod_name", pod.Name)
			err := client.Delete(context.TODO(), pod)
			if err != nil {
				errs = append(errs, err)
			}
		}(id)
	}
	wg.Wait()
	return errs
}

func manageUnscheduledPodNodes(pods []*corev1.Pod) []string {
	var output []string
	for _, pod := range pods {
		idcond, condition := podutils.GetPodCondition(&pod.Status, corev1.PodScheduled)
		if idcond == -1 {
			continue
		}
		if condition.Status == corev1.ConditionFalse && condition.Reason == corev1.PodReasonUnschedulable {
			nodeName := pod.Spec.NodeName
			if nodeName == "" {
				nodeName = podaffinity.GetNodeNameFromAffinity(pod.Spec.Affinity)
			}
			output = append(output, nodeName)
		}
	}
	return output
}

// annotateCanaryDeploymentWithReason annotates the Canary deployment with a reason
func annotateCanaryDeploymentWithReason(client client.Client, eds *datadoghqv1alpha1.ExtendedDaemonSet, valueKey string, reasonKey string, reason datadoghqv1alpha1.ExtendedDaemonSetStatusReason) error {
	newEds := eds.DeepCopy()
	if newEds.Annotations == nil {
		newEds.Annotations = make(map[string]string)
	}

	if value, ok := newEds.Annotations[valueKey]; ok {
		if value == valueTrue {
			return nil
		}
	}
	newEds.Annotations[valueKey] = valueTrue
	newEds.Annotations[reasonKey] = string(reason)

	if err := client.Update(context.TODO(), newEds); err != nil {
		return err
	}
	return nil
}

// pauseCanaryDeployment updates two annotations so that the Canary deployment is marked as paused, along with a reason
func pauseCanaryDeployment(client client.Client, eds *datadoghqv1alpha1.ExtendedDaemonSet, reason datadoghqv1alpha1.ExtendedDaemonSetStatusReason) error {
	return annotateCanaryDeploymentWithReason(
		client,
		eds,
		datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey,
		datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedReasonAnnotationKey,
		reason,
	)
}

// failCanaryDeployment updates two annotations so that the Canary deployment is marked as failed, along with a reason
func failCanaryDeployment(client client.Client, eds *datadoghqv1alpha1.ExtendedDaemonSet, reason datadoghqv1alpha1.ExtendedDaemonSetStatusReason) error {
	return annotateCanaryDeploymentWithReason(
		client,
		eds,
		datadoghqv1alpha1.ExtendedDaemonSetCanaryFailedAnnotationKey,
		datadoghqv1alpha1.ExtendedDaemonSetCanaryFailedReasonAnnotationKey,
		reason,
	)
}

// addPodLabel adds a given label to a pod, no-op if the pod is nil or if the label exists
func addPodLabel(c client.Client, pod *corev1.Pod, k, v string) error {
	if pod == nil {
		return nil
	}
	if label, found := pod.GetLabels()[k]; found && label == v {
		// The label is there, nothing to do
		return nil
	}
	pod.Labels[k] = v
	return c.Update(context.TODO(), pod)
}

// deletePodLabel deletes a given pod label, no-op if the pod is nil or if the label doesn't exists
func deletePodLabel(c client.Client, pod *corev1.Pod, k string) error {
	if pod == nil {
		return nil
	}
	if _, found := pod.GetLabels()[k]; !found {
		// The label is not there, nothing to do
		return nil
	}
	delete(pod.Labels, k)
	return c.Update(context.TODO(), pod)
}
