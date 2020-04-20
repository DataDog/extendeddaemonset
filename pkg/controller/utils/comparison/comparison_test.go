// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package comparison

import "testing"

func TestGenerateHashFromEDSResourceNodeAnnotation(t *testing.T) {
	type args struct {
		edsNamespace    string
		edsName         string
		nodeAnnotations map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no annotations",
			args: args{edsNamespace: "bar", edsName: "foo", nodeAnnotations: nil},
			want: "",
		},
		{
			name: "annotations present but not for this EDS",
			args: args{
				edsNamespace: "bar",
				edsName:      "foo",
				nodeAnnotations: map[string]string{
					"resources.extendeddaemonset.datadoghq.com/default.foo.daemons": "{\"limits\":{\"cpu\": \"1\"}}",
				},
			},
			want: "",
		},
		{
			name: "annotations present for this EDS",
			args: args{
				edsNamespace: "bar",
				edsName:      "foo",
				nodeAnnotations: map[string]string{
					"resources.extendeddaemonset.datadoghq.com/bar.foo.daemons": "{\"limits\":{\"cpu\": \"1\"}}",
				},
			},
			want: "bc9eacb89b7a44531492e87a37922dc3",
		},
		{
			name: "annotations present but not all for this EDS",
			args: args{
				edsNamespace: "bar",
				edsName:      "foo",
				nodeAnnotations: map[string]string{
					"resources.extendeddaemonset.datadoghq.com/default.foo.daemons": "{\"requests\":{\"cpu\": \"1\"}}",
					"resources.extendeddaemonset.datadoghq.com/bar.foo.daemons":     "{\"limits\":{\"cpu\": \"1\"}}",
				},
			},
			want: "bc9eacb89b7a44531492e87a37922dc3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateHashFromEDSResourceNodeAnnotation(tt.args.edsNamespace, tt.args.edsName, tt.args.nodeAnnotations); got != tt.want {
				t.Errorf("GenerateHashFromEDSResourceNodeAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}
