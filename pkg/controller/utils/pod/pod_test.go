// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package pod

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	ctrltest "github.com/DataDog/extendeddaemonset/pkg/controller/test"
)

func Test_HighestPodRestartCount(t *testing.T) {
	tests := []struct {
		name             string
		pod              *v1.Pod
		wantRestartCount int
		wantReason       datadoghqv1alpha1.ExtendedDaemonSetStatusReason
	}{
		{
			name: "restart count greater than max tolerable, due to CLB",
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
			wantRestartCount: 10,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			name: "restart count less than max tolerable",
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
			wantRestartCount: 4,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			name: "restart count equal to max tolerable, due to CLB",
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
			wantRestartCount: 5,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			name: "restart count greater than tolerable, due to OOM",
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
			wantRestartCount: 6,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonOOM,
		},
		{
			name: "no restarts",
			pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
				ContainerStatuses: []v1.ContainerStatus{
					{
						RestartCount:         0,
						LastTerminationState: v1.ContainerState{},
					},
				},
			},
			),
			wantRestartCount: 0,
			wantReason:       "",
		},
		{
			name: "multiple containers where one has high restart count",
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
			wantRestartCount: 10,
			wantReason:       datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restartCount, reason := HighestRestartCount(tt.pod)
			assert.Equal(t, tt.wantRestartCount, restartCount)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}

func Test_MostRecentRestart(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		pod        *v1.Pod
		wantTime   time.Time
		wantReason datadoghqv1alpha1.ExtendedDaemonSetStatusReason
	}{
		{
			name: "multiple restarts",
			pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
				ContainerStatuses: []v1.ContainerStatus{
					{
						RestartCount: 10,
						LastTerminationState: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								Reason:     "CrashLoopBackOff",
								FinishedAt: metav1.NewTime(now.Add(-time.Hour)),
							},
						},
					},
					{
						RestartCount: 1,
						LastTerminationState: v1.ContainerState{
							Terminated: &v1.ContainerStateTerminated{
								Reason:     "OOMKilled",
								FinishedAt: metav1.NewTime(now.Add(-2 * time.Hour)),
							},
						},
					},
				},
			},
			),
			wantTime:   now.Add(-time.Hour),
			wantReason: datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			name: "no restarts",
			pod: ctrltest.NewPod("bar", "pod1", "node1", &ctrltest.NewPodOptions{
				ContainerStatuses: []v1.ContainerStatus{
					{},
				},
			},
			),
			wantTime:   time.Time{},
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restartTime, reason := MostRecentRestart(tt.pod)
			assert.Equal(t, tt.wantTime, restartTime)
			assert.Equal(t, tt.wantReason, reason)
		})
	}
}

func TestIsPodAvailable(t *testing.T) {
	now := metav1.Now()

	type args struct {
		pod             *v1.Pod
		minReadySeconds int32
		now             metav1.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Not available, because not ready",
			args: args{
				pod:             newPod(now, false, 0),
				minReadySeconds: 0,
				now:             now,
			},
			want: false,
		},
		{
			name: "Not available, Pod ready, but not since enough time",
			args: args{
				pod:             newPod(now, true, 0),
				minReadySeconds: 1,
				now:             now,
			},
			want: false,
		},
		{
			name: "Available, Pod ready and minReady == 0",
			args: args{
				pod:             newPod(now, true, 0),
				minReadySeconds: 0,
				now:             now,
			},
			want: true,
		},
		{
			name: "Available, Pod ready since enough time",
			args: args{
				pod:             newPod(now, true, 51),
				minReadySeconds: 50,
				now:             now,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPodAvailable(tt.args.pod, tt.args.minReadySeconds, tt.args.now); got != tt.want {
				t.Errorf("IsPodAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newPod(now metav1.Time, ready bool, beforeSec int) *v1.Pod {
	conditionStatus := v1.ConditionFalse
	if ready {
		conditionStatus = v1.ConditionTrue
	}

	return &v1.Pod{
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:               v1.PodReady,
					LastTransitionTime: metav1.NewTime(now.Time.Add(time.Duration(-beforeSec) * time.Second)),
					Status:             conditionStatus,
				},
			},
		},
	}
}

func Test_convertReasonToEDSStatusReason(t *testing.T) {
	tests := []struct {
		reason string
		want   datadoghqv1alpha1.ExtendedDaemonSetStatusReason
	}{
		{
			reason: "CrashLoopBackOff",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCLB,
		},
		{
			reason: "OOMKilled",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonOOM,
		},
		{
			reason: "RestartsTimeoutExceeded",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusRestartsTimeoutExceeded,
		},
		{
			reason: "SlowStartTimeoutExceeded",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusSlowStartTimeoutExceeded,
		},
		{
			reason: "ErrImagePull",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonErrImagePull,
		},
		{
			reason: "ImagePullBackOff",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonImagePullBackOff,
		},
		{
			reason: "CreateContainerConfigError",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCreateContainerConfigError,
		},
		{
			reason: "CreateContainerError",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonCreateContainerError,
		},
		{
			reason: "PreStartHookError",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonPreStartHookError,
		},
		{
			reason: "PostStartHookError",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonPostStartHookError,
		},
		{
			reason: "does not exist",
			want:   datadoghqv1alpha1.ExtendedDaemonSetStatusReasonUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if got := convertReasonToEDSStatusReason(tt.reason); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertReasonToEDSStatusReason() = %v, want %v", got, tt.want)
			}
		})
	}
}
