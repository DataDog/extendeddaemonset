// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package v1alpha1

import "errors"

// ErrInvalidAutoFailRestarts is returned in case of a validation failure for maxRestarts in autoFail.
var ErrInvalidAutoFailRestarts = errors.New("canary autoFail.maxRestarts must be higher than autoPause.maxRestarts")

// ValidateExtendedDaemonSetSpec validates an ExtendedDaemonSet spec
// returns true if yes, else no.
func ValidateExtendedDaemonSetSpec(spec *ExtendedDaemonSetSpec) error {
	if canary := spec.Strategy.Canary; canary != nil {
		if *canary.AutoFail.Enabled && *canary.AutoPause.Enabled && *canary.AutoFail.MaxRestarts < *canary.AutoPause.MaxRestarts {
			return ErrInvalidAutoFailRestarts
		}
	}

	return nil
}
