// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package pod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	ctrltest "github.com/DataDog/extendeddaemonset/pkg/controller/test"
)

func Test_HighestPodRestartCount(t *testing.T) {
	type args struct {
		pod *v1.Pod
	}

	tests := []struct {
		name             string
		args             args
		wantRestartCount int
		wantReason       datadoghqv1alpha1.ExtendedDaemonSetStatusReason
	}{
		{
			name: "restart count greater than max tolerable, due to CLB",
			args: args{
				pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
					ContainerStatuses: []v1.ContainerStatus{
						{
							RestartCount: 10,
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
				),
			},
			wantRestartCount: 10,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			name: "restart count less than max tolerable",
			args: args{
				pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
					ContainerStatuses: []v1.ContainerStatus{
						{
							RestartCount: 4,
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
				),
			},
			wantRestartCount: 4,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			name: "restart count equal to max tolerable, due to CLB",
			args: args{
				pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
					ContainerStatuses: []v1.ContainerStatus{
						{
							RestartCount: 5,
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
				),
			},
			wantRestartCount: 5,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			name: "restart count greater than tolerable, due to OOM",
			args: args{
				pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
					ContainerStatuses: []v1.ContainerStatus{
						{
							RestartCount: 6,
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "OOMKilled",
								},
							},
						},
					},
				},
				),
			},
			wantRestartCount: 6,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonOOM,
		},
		{
			name: "no restarts",
			args: args{
				pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
					ContainerStatuses: []v1.ContainerStatus{
						{
							RestartCount:         0,
							LastTerminationState: v1.ContainerState{},
						},
					},
				},
				),
			},
			wantRestartCount: 0,
			wantReason:       "",
		},
		{
			name: "multiple containers where one has high restart count",
			args: args{
				pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
					ContainerStatuses: []v1.ContainerStatus{
						{
							RestartCount:         0,
							LastTerminationState: v1.ContainerState{},
						},
						{
							RestartCount: 10,
							LastTerminationState: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
				),
			},
			wantRestartCount: 10,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restartCount, reason := HighestRestartCount(tt.args.pod)
			assert.Equal(t, tt.wantRestartCount, restartCount)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}
