// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
)

func TestCalculateMaxCreation(t *testing.T) {
	now := time.Now()

	defaultParams := &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{}
	defaultParams = datadoghqv1alpha1.DefaultExtendedDaemonSetSpecStrategyRollingUpdate(defaultParams)
	type args struct {
		params      *datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate
		nbNodes     int
		rsStartTime time.Time
		now         time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "startTime",
			args: args{
				nbNodes:     100,
				now:         now,
				params:      defaultParams,
				rsStartTime: now,
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "2min later, with default strategy",
			args: args{
				nbNodes:     100,
				now:         now,
				params:      defaultParams,
				rsStartTime: now.Add(-2 * time.Minute),
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "2min later, with default strategy",
			args: args{
				nbNodes: 100,
				now:     now,
				params: datadoghqv1alpha1.DefaultExtendedDaemonSetSpecStrategyRollingUpdate(
					&datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
						SlowStartAdditiveIncrease: intstr.ValueOrDefault(nil, intstr.FromInt(2)),
					},
				),
				rsStartTime: now.Add(-2 * time.Minute),
			},
			want:    6,
			wantErr: false,
		},
		{
			name: "5min later, with default strategy",
			args: args{
				nbNodes: 100,
				now:     now,
				params: datadoghqv1alpha1.DefaultExtendedDaemonSetSpecStrategyRollingUpdate(
					&datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
						SlowStartAdditiveIncrease: intstr.ValueOrDefault(nil, intstr.FromInt(10)),
					},
				),
				rsStartTime: now.Add(-5 * time.Minute),
			},
			want:    60,
			wantErr: false,
		},
		{
			name: "value parse error",
			args: args{
				nbNodes: 100,
				now:     now,
				params: datadoghqv1alpha1.DefaultExtendedDaemonSetSpecStrategyRollingUpdate(
					&datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
						SlowStartAdditiveIncrease: intstr.ValueOrDefault(nil, intstr.FromString("10$")),
					},
				),
				rsStartTime: now.Add(-5 * time.Minute),
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "10min later, max // pods",
			args: args{
				nbNodes: 100,
				now:     now,
				params: datadoghqv1alpha1.DefaultExtendedDaemonSetSpecStrategyRollingUpdate(
					&datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
						SlowStartAdditiveIncrease: intstr.ValueOrDefault(nil, intstr.FromInt(30)),
					},
				),
				rsStartTime: now.Add(-10 * time.Minute),
			},
			want:    250,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculateMaxCreation(tt.args.params, tt.args.nbNodes, tt.args.rsStartTime, tt.args.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("calculateMaxCreation() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if got != tt.want {
				t.Errorf("calculateMaxCreation() = %v, want %v", got, tt.want)
			}
		})
	}
}
