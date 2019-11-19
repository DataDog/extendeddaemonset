// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package pod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
)

// CreatePodFromDaemonSetReplicaSet use to create a Pod from a ReplicaSet instance and a specific Node name.
func CreatePodFromDaemonSetReplicaSet(scheme *runtime.Scheme, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, nodeName string, addNodeAffinity bool) *corev1.Pod {

	templateCopy := replicaset.Spec.Template.DeepCopy()
	{
		templateCopy.ObjectMeta.Namespace = replicaset.Namespace
		templateCopy.ObjectMeta.GenerateName = fmt.Sprintf("%s-", replicaset.Name)
	}

	if templateCopy.ObjectMeta.Labels == nil {
		templateCopy.ObjectMeta.Labels = map[string]string{}
	}
	templateCopy.ObjectMeta.Labels[datadoghqv1alpha1.ExtendedDaemonSetReplicaSetNameLabelKey] = replicaset.Name
	templateCopy.ObjectMeta.Labels[datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey] = replicaset.Labels[datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey]

	if templateCopy.ObjectMeta.Annotations == nil {
		templateCopy.ObjectMeta.Annotations = map[string]string{}
	}
	templateCopy.ObjectMeta.Annotations[datadoghqv1alpha1.MD5ExtendedDaemonSetAnnotationKey] = replicaset.Spec.TemplateGeneration
	templateCopy.ObjectMeta.Annotations[DaemonsetClusterAutoscalerPodAnnotationKey] = "true"

	pod := &corev1.Pod{
		ObjectMeta: templateCopy.ObjectMeta,
		Spec:       templateCopy.Spec,
	}
	if nodeName != "" {
		pod.Spec.NodeName = nodeName

		if addNodeAffinity {
			pod.Spec.Affinity = ReplaceNodeNameNodeAffinity(pod.Spec.Affinity, nodeName)
		}
	}

	if scheme != nil {
		_ = controllerutil.SetControllerReference(replicaset, pod, scheme)
	}

	return pod
}
