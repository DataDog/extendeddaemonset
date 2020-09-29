// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package testutils

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	"sigs.k8s.io/controller-runtime/pkg/client"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
)

// UpdateExtendedDaemonSetFunc used to update a ExtendedDaemonSet with retry and timeout policy
func UpdateExtendedDaemonSetFunc(client client.Client, namespace, name string, updateFunc func(eds *datadoghqv1alpha1.ExtendedDaemonSet), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, eds); err != nil {
			return false, nil
		}

		updateEds := eds.DeepCopy()
		updateFunc(updateEds)
		if err := client.Update(context.TODO(), updateEds); err != nil {
			return false, err
		}
		return true, nil
	})

}

// UpdateDaemonSetFunc used to update a DaemonSet with retry and timeout policy
func UpdateDaemonSetFunc(client client.Client, namespace, name string, updateFunc func(ds *appsv1.DaemonSet), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		ds := &appsv1.DaemonSet{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, ds); err != nil {
			return false, nil
		}

		updateDs := ds.DeepCopy()
		updateFunc(updateDs)
		if err := client.Update(context.TODO(), updateDs); err != nil {
			return false, err
		}
		return true, nil
	})

}
