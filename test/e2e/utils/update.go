// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package utils

import (
	goctx "context"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
)

// UpdateExtendedDaemonSetFunc used to update a ExtendedDaemonSet with retry and timeout policy
func UpdateExtendedDaemonSetFunc(f *framework.Framework, namespace, name string, updateFunc func(eds *datadoghqv1alpha1.ExtendedDaemonSet), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		eds := &datadoghqv1alpha1.ExtendedDaemonSet{}
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, eds); err != nil {
			return false, nil
		}

		updateEds := eds.DeepCopy()
		updateFunc(updateEds)
		if err := f.Client.Update(goctx.TODO(), updateEds); err != nil {
			return false, err
		}
		return true, nil
	})

}

// UpdateDaemonSetFunc used to update a DaemonSet with retry and timeout policy
func UpdateDaemonSetFunc(f *framework.Framework, namespace, name string, updateFunc func(ds *appsv1.DaemonSet), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		ds := &appsv1.DaemonSet{}
		if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, ds); err != nil {
			return false, nil
		}

		updateDs := ds.DeepCopy()
		updateFunc(updateDs)
		if err := f.Client.Update(goctx.TODO(), updateDs); err != nil {
			return false, err
		}
		return true, nil
	})

}
