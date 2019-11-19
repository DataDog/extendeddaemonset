// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package utils

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	framework "github.com/operator-framework/operator-sdk/pkg/test"

	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
)

// WaitForFuncOnExtendedDaemonset used to wait a valid condition on a ExtendedDaemonSet
func WaitForFuncOnExtendedDaemonset(t *testing.T, client framework.FrameworkClient, namespace, name string, f func(dd *datadoghqv1alpha1.ExtendedDaemonSet) (bool, error), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		objKey := dynclient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		extendeddaemonset := &datadoghqv1alpha1.ExtendedDaemonSet{}
		err := client.Get(context.TODO(), objKey, extendeddaemonset)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s ExtendedDaemonSet\n", name)
				return false, nil
			}
			return false, err
		}

		ok, err := f(extendeddaemonset)
		t.Logf("Waiting for condition function to be true ok for %s ExtendedDaemonSet (%t/%v)\n", name, ok, err)
		return ok, err
	})
}

// WaitForFuncOnExtendedDaemonsetReplicaSet used to wait a valid condition on a ExtendedDaemonSet
func WaitForFuncOnExtendedDaemonsetReplicaSet(t *testing.T, client framework.FrameworkClient, namespace, name string, f func(rs *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (bool, error), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		objKey := dynclient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		extendeddaemonsetreplicaset := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}
		err := client.Get(context.TODO(), objKey, extendeddaemonsetreplicaset)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s ExtendedDaemonSetReplicaSet \n", name)
				return false, nil
			}
			return false, err
		}

		ok, err := f(extendeddaemonsetreplicaset)
		t.Logf("Waiting for condition function to be true ok for %s ExtendedDaemonSetReplicaSet (%t/%v)\n", name, ok, err)
		return ok, err
	})
}

// WaitForFuncOnDaemonset used to wait a valid condition on a DaemonSet
func WaitForFuncOnDaemonset(t *testing.T, client framework.FrameworkClient, namespace, name string, f func(dd *appsv1.DaemonSet) (bool, error), retryInterval, timeout time.Duration) error {
	return wait.Poll(retryInterval, timeout, func() (bool, error) {
		objKey := dynclient.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}
		daemonset := &appsv1.DaemonSet{}
		err := client.Get(context.TODO(), objKey, daemonset)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s DaemonSet\n", name)
				return false, nil
			}
			return false, err
		}

		ok, err := f(daemonset)
		t.Logf("Waiting for condition function to be true ok for %s DaemonSet (%t/%v)\n", name, ok, err)
		return ok, err
	})
}
