// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package scheduler

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ctrltest "github.com/DataDog/extendeddaemonset/pkg/controller/test"
	"github.com/DataDog/extendeddaemonset/pkg/controller/utils/pod"
)

func TestCheckNodeFitness(t *testing.T) {
	now := time.Now()
	logf.SetLogger(zap.New())
	log := logf.Log.WithName("TestCheckNodeFitness")

	nodeReadyOptions := &ctrltest.NewNodeOptions{
		Labels: map[string]string{"app": "foo"},
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
	}
	nodeKOOptions := &ctrltest.NewNodeOptions{
		Labels: map[string]string{"app": "foo"},
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionFalse,
			},
		},
		Taints: []corev1.Taint{
			{
				Key:    "node.kubernetes.io/not-ready",
				Effect: corev1.TaintEffectNoExecute,
			},
		},
	}
	nodeUnscheduledOptions := &ctrltest.NewNodeOptions{
		Labels:        map[string]string{"app": "foo"},
		Unschedulable: true,
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
		Taints: []corev1.Taint{
			{
				Key:    "node.kubernetes.io/unschedulable",
				Effect: corev1.TaintEffectNoSchedule,
			},
		},
	}
	nodeTaintedOptions := &ctrltest.NewNodeOptions{
		Labels: map[string]string{"app": "foo"},
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
		Taints: []corev1.Taint{
			{
				Key:    "mytaint",
				Effect: corev1.TaintEffectNoSchedule,
			},
		},
	}
	node1 := ctrltest.NewNode("node1", nodeReadyOptions)
	node2 := ctrltest.NewNode("node2", nodeKOOptions)
	node3 := ctrltest.NewNode("node3", nodeUnscheduledOptions)
	node4 := ctrltest.NewNode("node4", nodeTaintedOptions)

	pod1 := ctrltest.NewPod("foo", "pod1", "", &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now),
		NodeSelector:      map[string]string{"app": "foo"},
		Tolerations:       pod.StandardDaemonSetTolerations,
	})

	type args struct {
		pod  *corev1.Pod
		node *corev1.Node
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "node ready",
			args: args{
				pod:  pod1,
				node: node1,
			},
			want: true,
		},
		{
			name: "node not ready",
			args: args{
				pod:  pod1,
				node: node2,
			},
			want: true,
		},
		{
			name: "node unschedulable",
			args: args{
				pod:  pod1,
				node: node3,
			},
			want: true,
		},
		{
			name: "node tainted",
			args: args{
				pod:  pod1,
				node: node4,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckNodeFitness(log.WithName(tt.name), tt.args.pod, tt.args.node); got != tt.want {
				t.Errorf("CheckNodeFitness() = %v, want %v", got, tt.want)
			}
		})
	}
}
