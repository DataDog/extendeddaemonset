// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package pod

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils/affinity"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils/comparison"
)

// CreatePodFromDaemonSetReplicaSet use to create a Pod from a ReplicaSet instance and a specific Node name.
func CreatePodFromDaemonSetReplicaSet(scheme *runtime.Scheme, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, node *corev1.Node, addNodeAffinity bool) (*corev1.Pod, error) {
	var err error
	templateCopy := replicaset.Spec.Template.DeepCopy()
	{
		templateCopy.ObjectMeta.Namespace = replicaset.Namespace
		templateCopy.ObjectMeta.GenerateName = fmt.Sprintf("%s-", replicaset.Name)
	}

	if templateCopy.ObjectMeta.Labels == nil {
		templateCopy.ObjectMeta.Labels = map[string]string{}
	}
	templateCopy.ObjectMeta.Labels[datadoghqv1alpha1.ExtendedDaemonSetReplicaSetNameLabelKey] = replicaset.Name
	edsName := replicaset.Labels[datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey]
	templateCopy.ObjectMeta.Labels[datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey] = edsName

	if templateCopy.ObjectMeta.Annotations == nil {
		templateCopy.ObjectMeta.Annotations = map[string]string{}
	}
	templateCopy.ObjectMeta.Annotations[datadoghqv1alpha1.MD5ExtendedDaemonSetAnnotationKey] = replicaset.Spec.TemplateGeneration
	templateCopy.ObjectMeta.Annotations[DaemonsetClusterAutoscalerPodAnnotationKey] = "true"

	if node != nil {
		err = overwriteResourcesFromNode(templateCopy, replicaset.Namespace, edsName, node)
		templateCopy.ObjectMeta.Annotations[datadoghqv1alpha1.MD5NodeExtendedDaemonSetAnnotationKey] = comparison.GenerateHashFromEDSResourceNodeAnnotation(replicaset.Namespace, edsName, node.Annotations)
	}

	pod := &corev1.Pod{
		ObjectMeta: templateCopy.ObjectMeta,
		Spec:       templateCopy.Spec,
	}
	if node != nil {
		pod.Spec.NodeName = node.Name

		if addNodeAffinity {
			pod.Spec.Affinity = affinity.ReplaceNodeNameNodeAffinity(pod.Spec.Affinity, node.Name)
		}
	}

	if scheme != nil {
		err = controllerutil.SetControllerReference(replicaset, pod, scheme)
	}

	return pod, err
}

func overwriteResourcesFromNode(template *corev1.PodTemplateSpec, edsNamespace, edsName string, node *corev1.Node) error {
	if node == nil {
		return nil
	}

	var errs []error
	for id, container := range template.Spec.Containers {
		ressourceAnnotationKey := fmt.Sprintf(datadoghqv1alpha1.ExtendedDaemonSetRessourceNodeAnnotationKey, edsNamespace, edsName, container.Name)
		if val, ok := node.GetAnnotations()[ressourceAnnotationKey]; ok {
			var newResources corev1.ResourceRequirements
			if err := json.Unmarshal([]byte(val), &newResources); err != nil {
				errWrap := fmt.Errorf("unable to decode %s annotation value, err: %v", ressourceAnnotationKey, err)
				errs = append(errs, errWrap)
				continue
			}
			template.Spec.Containers[id].Resources = newResources
		}
	}
	return errors.NewAggregate(errs)
}
