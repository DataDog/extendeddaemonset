// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package test

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
)

var (
	// apiVersion datadoghqv1alpha1 api version
	apiVersion = fmt.Sprintf("%s/%s", datadoghqv1alpha1.SchemeGroupVersion.Group, datadoghqv1alpha1.SchemeGroupVersion.Version)
)

// NewExtendedDaemonSetOptions set of option for the ExtendedDaemonset creation
type NewExtendedDaemonSetOptions struct {
	CreationTime  *time.Time
	Annotations   map[string]string
	Labels        map[string]string
	RollingUpdate *datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate
	Canary        *datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary
	Status        *datadoghqv1alpha1.ExtendedDaemonSetStatus
}

// NewExtendedDaemonSet return new ExtendedDDaemonset instance for test purpose
func NewExtendedDaemonSet(ns, name string, options *NewExtendedDaemonSetOptions) *datadoghqv1alpha1.ExtendedDaemonSet {
	dd := &datadoghqv1alpha1.ExtendedDaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExtendedDaemonSet",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       ns,
			Name:            name,
			Labels:          map[string]string{},
			Annotations:     map[string]string{},
			ResourceVersion: "1",
		},
	}
	if options != nil {
		if options.CreationTime != nil {
			dd.CreationTimestamp = metav1.NewTime(*options.CreationTime)
		}
		if options.Annotations != nil {
			for key, value := range options.Annotations {
				dd.Annotations[key] = value
			}
		}
		if options.Labels != nil {
			for key, value := range options.Labels {
				dd.Labels[key] = value
			}
		}
		if options.RollingUpdate != nil {
			dd.Spec.Strategy.RollingUpdate = *options.RollingUpdate
		}
		if options.Canary != nil {
			dd.Spec.Strategy.Canary = options.Canary
		}
		if options.Status != nil {
			dd.Status = *options.Status
		}
	}

	return dd
}

// NewExtendedDaemonSetReplicaSetOptions set of option for the ExtendedDaemonsetReplicaSet creation
type NewExtendedDaemonSetReplicaSetOptions struct {
	CreationTime *time.Time
	Annotations  map[string]string
	Labels       map[string]string
	GenerateName string
	OwnerRefName string
	Status       *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus
}

// NewExtendedDaemonSetReplicaSet returns new ExtendedDaemonSetReplicaSet instance for testing purpose
func NewExtendedDaemonSetReplicaSet(ns, name string, options *NewExtendedDaemonSetReplicaSetOptions) *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet {
	dd := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExtendedDaemonSetReplicaSet",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ns,
			Name:        name,
			Labels:      map[string]string{},
			Annotations: map[string]string{},
		},
	}
	if options != nil {
		if options.GenerateName != "" {
			dd.GenerateName = options.GenerateName
		}
		if options.CreationTime != nil {
			dd.CreationTimestamp = metav1.NewTime(*options.CreationTime)
		}
		if options.Annotations != nil {
			for key, value := range options.Annotations {
				dd.Annotations[key] = value
			}
		}
		if options.Labels != nil {
			for key, value := range options.Labels {
				dd.Labels[key] = value
			}
		}
		if options.OwnerRefName != "" {
			dd.OwnerReferences = []metav1.OwnerReference{
				{
					Name: options.OwnerRefName,
					Kind: "ExtendedDaemonSet",
				},
			}
		}
		if options.Status != nil {
			dd.Status = *options.Status
		}
	}

	return dd
}
