// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func Test_compareSpecTemplateMD5Hash(t *testing.T) {
	type args struct {
		hash string
		pod  *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareSpecTemplateMD5Hash(tt.args.hash, tt.args.pod); got != tt.want {
				t.Errorf("compareSpecTemplateMD5Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}
