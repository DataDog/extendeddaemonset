// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Package config contains EDS controller configuration.
package config

import (
	"os"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

const (
	// NodeAffinityMatchSupportEnvVar use to know if the scheduler support this feature:
	// https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/#scheduled-by-default-scheduler-enabled-by-default-since-1-12.
	NodeAffinityMatchSupportEnvVar = "EDS_NODEAFFINITYMATCH_SUPPORT"

	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"
	// ValidationModeEnvVar is the constant for env variable EDS_VALIDATION_MODE
	// It allows to override default validationMode setting for ExtendedDaemonSetSpecStrategyCanary.
	ValidationModeEnvVar = "EDS_VALIDATION_MODE"
)

// GetWatchNamespaces returns the Namespaces the operator should be watching for changes.
func GetWatchNamespaces() []string {
	ns, found := os.LookupEnv(WatchNamespaceEnvVar)
	if !found {
		return nil
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	if strings.Contains(ns, ",") {
		return strings.Split(ns, ",")
	}

	return []string{ns}
}

// ManagerOptionsWithNamespaces returns an updated Options with namespaces information.
func ManagerOptionsWithNamespaces(logger logr.Logger, opt ctrl.Options) ctrl.Options {
	namespaces := GetWatchNamespaces()
	switch {
	case len(namespaces) == 0:
		logger.Info("Manager will watch and manage resources in all namespaces")
	case len(namespaces) == 1:
		logger.Info("Manager will be watching namespace", namespaces[0])
		opt.Cache.DefaultNamespaces = map[string]cache.Config{namespaces[0]: {}}
	case len(namespaces) > 1:
		// configure cluster-scoped with MultiNamespacedCacheBuilder
		logger.Info("Manager will be watching multiple namespaces", namespaces)
		opt.NewCache = func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			// https://github.com/kubernetes-sigs/controller-runtime/commit/3e35cab1ea29145d8699259b079633a2b8cfc116
			nsConfigs := make(map[string]cache.Config)
			for _, ns := range namespaces {
				nsConfigs[ns] = cache.Config{}
			}
			opts.DefaultNamespaces = nsConfigs
			return cache.New(config, opts)
		}
	}

	return opt
}
