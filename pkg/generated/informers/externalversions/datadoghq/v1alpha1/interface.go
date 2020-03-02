// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/datadog/extendeddaemonset/pkg/generated/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ExtendedDaemonSets returns a ExtendedDaemonSetInformer.
	ExtendedDaemonSets() ExtendedDaemonSetInformer
	// ExtendedDaemonSetReplicaSets returns a ExtendedDaemonSetReplicaSetInformer.
	ExtendedDaemonSetReplicaSets() ExtendedDaemonSetReplicaSetInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// ExtendedDaemonSets returns a ExtendedDaemonSetInformer.
func (v *version) ExtendedDaemonSets() ExtendedDaemonSetInformer {
	return &extendedDaemonSetInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}

// ExtendedDaemonSetReplicaSets returns a ExtendedDaemonSetReplicaSetInformer.
func (v *version) ExtendedDaemonSetReplicaSets() ExtendedDaemonSetReplicaSetInformer {
	return &extendedDaemonSetReplicaSetInformer{factory: v.factory, namespace: v.namespace, tweakListOptions: v.tweakListOptions}
}
