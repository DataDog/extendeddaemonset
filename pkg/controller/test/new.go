// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package test

import (
	corev1 "k8s.io/api/core/v1"

	utilaffinity "github.com/datadog/extendeddaemonset/pkg/controller/utils/affinity"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewNodeOptions store NewNode options
type NewNodeOptions struct {
	Annotations map[string]string
	Labels      map[string]string
	Conditions  []corev1.NodeCondition
}

// NewNode returns new node instance
func NewNode(name string, opts *NewNodeOptions) *corev1.Node {
	node := &corev1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
	}

	if opts != nil {
		if opts.Annotations != nil {
			for key, value := range opts.Annotations {
				node.Annotations[key] = value
			}
		}
		if opts.Labels != nil {
			for key, value := range opts.Labels {
				node.Labels[key] = value
			}
		}
		node.Status.Conditions = append(node.Status.Conditions, opts.Conditions...)
	}
	return node
}

// NewPodOptions store NewPod options
type NewPodOptions struct {
	CreationTimestamp metav1.Time
	Annotations       map[string]string
	Labels            map[string]string
	Phase             corev1.PodPhase
	Resources         corev1.ResourceRequirements
}

// NewPod used to return new pod instance
func NewPod(namespace, name, nodeName string, opts *NewPodOptions) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Affinity: &corev1.Affinity{},
			Containers: []corev1.Container{
				{
					Name: name,
				},
			},
		},
	}
	if opts != nil {
		pod.CreationTimestamp = opts.CreationTimestamp

		pod.Spec.Containers[0].Resources = opts.Resources

		if opts.Annotations != nil {
			for key, value := range opts.Annotations {
				pod.Annotations[key] = value
			}
		}
		if opts.Labels != nil {
			for key, value := range opts.Labels {
				pod.Labels[key] = value
			}
		}
		pod.Status.Phase = opts.Phase
	}

	utilaffinity.ReplaceNodeNameNodeAffinity(pod.Spec.Affinity, nodeName)
	return pod
}
