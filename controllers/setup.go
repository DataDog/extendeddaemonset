// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package controllers

import (
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonset"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetsetting"
)

// SetupControllers start all controllers (also used by unit and e2e tests).
func SetupControllers(mgr manager.Manager, nodeAffinityMatchSupport bool) error {
	if err := (&ExtendedDaemonSetReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ExtendedDaemonSet"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("ExtendedDaemonSet"),
		Options:  extendeddaemonset.ReconcilerOptions{},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller ExtendedDaemonSet: %w", err)
	}

	if err := (&ExtendedDaemonsetSettingReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ExtendedDaemonsetSetting"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("ExtendedDaemonsetSetting"),
		Options:  extendeddaemonsetsetting.ReconcilerOptions{},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller ExtendedDaemonsetSetting: %w", err)
	}

	if err := (&ExtendedDaemonSetReplicaSetReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ExtendedDaemonSetReplicaSet"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("ExtendedDaemonSetReplicaSet"),
		Options: extendeddaemonsetreplicaset.ReconcilerOptions{
			IsNodeAffinitySupported: nodeAffinityMatchSupport,
		},
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller ExtendedDaemonSetReplicaSet: %w", err)
	}

	return nil
}
