// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateExtendedDaemonSetSpec(t *testing.T) {
	validNoCanary := DefaultExtendedDaemonSetSpec(&ExtendedDaemonSetSpec{})
	validWithCanary := DefaultExtendedDaemonSetSpec(&ExtendedDaemonSetSpec{
		Strategy: ExtendedDaemonSetSpecStrategy{
			Canary: &ExtendedDaemonSetSpecStrategyCanary{},
		},
	})

	validAutoFail := validWithCanary.DeepCopy()
	*validAutoFail.Strategy.Canary.AutoPause.Enabled = true
	*validAutoFail.Strategy.Canary.AutoPause.MaxRestarts = 2
	*validAutoFail.Strategy.Canary.AutoFail.Enabled = true
	*validAutoFail.Strategy.Canary.AutoFail.MaxRestarts = 3

	invalidAutoFail := validWithCanary.DeepCopy()
	*invalidAutoFail.Strategy.Canary.AutoPause.Enabled = true
	*invalidAutoFail.Strategy.Canary.AutoPause.MaxRestarts = 2
	*invalidAutoFail.Strategy.Canary.AutoFail.Enabled = true
	*invalidAutoFail.Strategy.Canary.AutoFail.MaxRestarts = 1

	validAutoFailNoAutoPause := validWithCanary.DeepCopy()
	*validAutoFailNoAutoPause.Strategy.Canary.AutoPause.Enabled = false
	*validAutoFailNoAutoPause.Strategy.Canary.AutoPause.MaxRestarts = 2
	*validAutoFailNoAutoPause.Strategy.Canary.AutoFail.Enabled = true
	*validAutoFailNoAutoPause.Strategy.Canary.AutoFail.MaxRestarts = 2

	validAutoPauseNoAutoFail := validWithCanary.DeepCopy()
	*validAutoPauseNoAutoFail.Strategy.Canary.AutoPause.Enabled = true
	*validAutoPauseNoAutoFail.Strategy.Canary.AutoPause.MaxRestarts = 1
	*validAutoPauseNoAutoFail.Strategy.Canary.AutoFail.Enabled = false
	*validAutoPauseNoAutoFail.Strategy.Canary.AutoFail.MaxRestarts = 1

	tests := []struct {
		name string
		spec *ExtendedDaemonSetSpec
		err  error
	}{
		{
			name: "valid no canary",
			spec: validNoCanary,
		},
		{
			name: "valid with canary",
			spec: validWithCanary,
		},
		{
			name: "valid autoFail maxRestarts",
			spec: validAutoFail,
		},
		{
			name: "invalid autoFail maxRestarts",
			spec: invalidAutoFail,
			err:  ErrInvalidAutoFailRestarts,
		},
		{
			name: "valid autoFail no autoPause",
			spec: validAutoFailNoAutoPause,
		},
		{
			name: "valid autoPause no autoFail",
			spec: validAutoFailNoAutoPause,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateExtendedDaemonSetSpec(test.spec)
			assert.Equal(t, test.err, err)
		})
	}
}
