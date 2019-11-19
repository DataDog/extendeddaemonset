// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package config

const (
	// NodeAffinityMatchSupportEnvVar use to know if the scheduler support this feature:
	// https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/#scheduled-by-default-scheduler-enabled-by-default-since-1-12
	NodeAffinityMatchSupportEnvVar = "EDS_NODEAFFINITYMATCH_SUPPORT"
)
