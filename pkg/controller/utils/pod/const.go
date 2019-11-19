// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package pod

const (
	// DaemonsetClusterAutoscalerPodAnnotationKey use to inform the cluster-autoscaler that a pod
	// should be considered as a DaemonSet pod
	DaemonsetClusterAutoscalerPodAnnotationKey = "cluster-autoscaler.kubernetes.io/daemonset-pod"
)
