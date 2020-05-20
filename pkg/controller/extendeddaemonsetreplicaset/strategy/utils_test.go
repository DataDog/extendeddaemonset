// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1/test"
	commontest "github.com/datadog/extendeddaemonset/pkg/controller/test"
)

func Test_compareWithExtendedDaemonsetSettingOverwrite(t *testing.T) {
	nodeName1 := "node1"
	nodeOptions := &commontest.NewNodeOptions{}
	node1 := commontest.NewNode(nodeName1, nodeOptions)

	resource1 := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			"cpu": resource.MustParse("1"),
		},
	}
	pod1Option := &commontest.NewPodOptions{Resources: *resource1}
	pod1 := commontest.NewPod("bar", "pod1", nodeName1, pod1Option)
	pod1.Spec.Containers[0].Resources = *resource1

	edsNode1Options := &test.NewExtendedDaemonsetSettingOptions{
		Resources: map[string]corev1.ResourceRequirements{
			"pod1": *resource1,
		},
	}
	extendedDaemonsetSetting1 := test.NewExtendedDaemonsetSetting("bar", "foo", "foo", edsNode1Options)

	edsNode2Options := &test.NewExtendedDaemonsetSettingOptions{
		Resources: map[string]corev1.ResourceRequirements{
			"pod1": {
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse("2"),
					"memory": resource.MustParse("1G"),
				},
			},
		},
	}
	extendedDaemonsetSetting2 := test.NewExtendedDaemonsetSetting("bar", "foo", "foo", edsNode2Options)

	type args struct {
		pod  *corev1.Pod
		node *NodeItem
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "empty ExtendedDaemonsetSetting",
			args: args{
				pod: pod1,
				node: &NodeItem{
					Node: node1,
				},
			},
			want: true,
		},
		{
			name: "ExtendedDaemonsetSetting that match",
			args: args{
				pod: pod1,
				node: &NodeItem{
					Node:                     node1,
					ExtendedDaemonsetSetting: extendedDaemonsetSetting1,
				},
			},
			want: true,
		},
		{
			name: "ExtendedDaemonsetSetting doesn't match",
			args: args{
				pod: pod1,
				node: &NodeItem{
					Node:                     node1,
					ExtendedDaemonsetSetting: extendedDaemonsetSetting2,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareWithExtendedDaemonsetSettingOverwrite(tt.args.pod, tt.args.node); got != tt.want {
				t.Errorf("compareWithExtendedDaemonsetSettingOverwrite() = %v, want %v", got, tt.want)
			}
		})
	}
}
