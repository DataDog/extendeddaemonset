// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonset

import (
	"testing"
	"time"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsCanaryPhaseEnded(t *testing.T) {
	now := time.Now()
	type args struct {
		specCanary *datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary
		rs         *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
		now        time.Time
	}
	tests := []struct {
		name         string
		args         args
		want         bool
		wantDuration time.Duration
	}{
		{
			name: "not spec == nil",
			args: args{
				specCanary: nil,
				rs:         &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{},
				now:        now,
			},
			want: true,
		},
		{
			name: "not canary not done",
			args: args{
				specCanary: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					Duration: &metav1.Duration{Duration: time.Hour},
				},
				rs: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.NewTime(now.Add(-time.Minute)),
					},
				},
				now: now,
			},
			want:         false,
			wantDuration: 59 * time.Minute,
		},
		{
			name: "not canary duration not set",
			args: args{
				specCanary: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{},
				rs: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.NewTime(now.Add(-time.Minute)),
					},
				},
				now: now,
			},
			want: false,
		},
		{
			name: "canary paused duration exceeded",
			args: args{
				specCanary: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					Duration: &metav1.Duration{Duration: time.Hour},
					Paused:   true,
				},
				rs: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					},
				},
				now: now,
			},
			want: false,
		},
		{
			name: "not canary done",
			args: args{
				specCanary: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					Duration: &metav1.Duration{Duration: time.Hour},
				},
				rs: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					},
				},
				now: now,
			},
			want:         true,
			wantDuration: -time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotDuration := IsCanaryPhaseEnded(tt.args.specCanary, tt.args.rs, tt.args.now)
			if got != tt.want {
				t.Errorf("IsCanaryPhaseEnded() = %v, want %v", got, tt.want)
			}
			if gotDuration != tt.wantDuration {
				t.Errorf("IsCanaryPhaseEnded() = %v, wantDuration %v", gotDuration, tt.wantDuration)
			}
		})
	}
}

func TestIsCanaryDeploymentValid(t *testing.T) {
	type args struct {
		dsAnnotations map[string]string
		rsName        string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "annotation found - correct rs name",
			args: args{
				dsAnnotations: map[string]string{
					"extendeddaemonset.datadoghq.com/canary-valid": "rsName",
				},
				rsName: "rsName",
			},
			want: true,
		},
		{
			name: "annotation found - incorrect rs name",
			args: args{
				dsAnnotations: map[string]string{
					"extendeddaemonset.datadoghq.com/canary-valid": "rsName",
				},
				rsName: "anotherRsName",
			},
			want: false,
		},
		{
			name: "annotation not found",
			args: args{
				dsAnnotations: map[string]string{
					"extendeddaemonset.datadoghq.com/another-annotation": "rsName",
				},
				rsName: "rsName",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCanaryDeploymentValid(tt.args.dsAnnotations, tt.args.rsName); got != tt.want {
				t.Errorf("IsCanaryDeploymentValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
