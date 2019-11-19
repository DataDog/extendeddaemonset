// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

import (
	"reflect"
	"testing"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	datadoghqv1alpha1test "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1/test"
	ctrltest "github.com/datadog/extendeddaemonset/pkg/controller/test"
)

func TestFilterPodsByNode(t *testing.T) {
	now := time.Now()
	ns := "foo"
	NodeNameA := "nodeA"
	NodeNameB := "nodeB"
	pod1NodeA := ctrltest.NewPod(ns, "pod1", NodeNameA, &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now),
	})
	pod2NodeB := ctrltest.NewPod(ns, "pod2", NodeNameB, &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now),
	})
	pod3NodeA := ctrltest.NewPod(ns, "pod3", NodeNameA, &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now.Truncate(time.Minute)),
	})
	tests := []struct {
		name           string
		podsByNodeName map[string][]*corev1.Pod
		want           map[string]*corev1.Pod
		want1          []*corev1.Pod
	}{
		{
			name: "one node, one pod",
			podsByNodeName: map[string][]*corev1.Pod{
				NodeNameA: {pod1NodeA},
			},
			want: map[string]*corev1.Pod{
				NodeNameA: pod1NodeA,
			},
			want1: []*corev1.Pod{},
		},
		{
			name: "2 nodes, 2 pods",
			podsByNodeName: map[string][]*corev1.Pod{
				NodeNameA: {pod1NodeA},
				NodeNameB: {pod2NodeB},
			},
			want: map[string]*corev1.Pod{
				NodeNameA: pod1NodeA,
				NodeNameB: pod2NodeB,
			},
			want1: []*corev1.Pod{},
		},
		{
			name: "2 nodes, 3 pods",
			podsByNodeName: map[string][]*corev1.Pod{
				NodeNameA: {pod1NodeA, pod3NodeA},
				NodeNameB: {pod2NodeB},
			},
			want: map[string]*corev1.Pod{
				NodeNameA: pod3NodeA,
				NodeNameB: pod2NodeB,
			},
			want1: []*corev1.Pod{pod1NodeA},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := FilterPodsByNode(tt.podsByNodeName)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterPodsByNode() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("FilterPodsByNode() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestFilterAndMapPodsByNode(t *testing.T) {
	now := time.Now()
	logf.SetLogger(logf.ZapLogger(true))
	log := logf.Log.WithName("TestFilterAndMapPodsByNode")

	ns := "foo"
	nodeReadyOptions := &ctrltest.NewNodeOptions{
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
	}
	nodeKOOptions := &ctrltest.NewNodeOptions{
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionFalse,
			},
		},
	}
	node1 := ctrltest.NewNode("node1", nodeReadyOptions)
	node2 := ctrltest.NewNode("node2", nodeReadyOptions)
	node3 := ctrltest.NewNode("node3", nodeReadyOptions)
	node4 := ctrltest.NewNode("node4", nodeKOOptions)

	pod1Node1 := ctrltest.NewPod(ns, "pod1", node1.Name, &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now),
	})
	pod2Node2 := ctrltest.NewPod(ns, "pod2", node2.Name, &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now),
	})
	pod3Node3 := ctrltest.NewPod(ns, "pod3", node3.Name, &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now.Truncate(time.Minute)),
	})

	pod4Node1 := ctrltest.NewPod(ns, "pod4", node1.Name, &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now),
		Phase:             corev1.PodUnknown,
	})

	pod3NodeFake := ctrltest.NewPod(ns, "pod3", "fakenode", &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now.Truncate(time.Minute)),
	})

	pod3NodeFakeBis := ctrltest.NewPod(ns, "pod3", "fakenode", &ctrltest.NewPodOptions{
		CreationTimestamp: metav1.NewTime(now.Truncate(time.Minute)),
	})
	metaNow := metav1.NewTime(now)
	pod3NodeFakeBis.DeletionTimestamp = &metaNow

	type args struct {
		replicaset  *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
		nodeList    *corev1.NodeList
		podList     *corev1.PodList
		ignoreNodes []string
	}
	tests := []struct {
		name                string
		args                args
		wantPodByNode       map[string]*corev1.Pod
		wantPodToDelete     []*corev1.Pod
		wantUnscheduledPods []*corev1.Pod
	}{
		{
			name: "one pod, one filtered node",
			args: args{
				replicaset: datadoghqv1alpha1test.NewExtendedDaemonSetReplicaSet("foo", "bar", nil),
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node1, *node2, *node3},
				},
				podList: &corev1.PodList{
					Items: []corev1.Pod{
						*pod1Node1,
					},
				},
				ignoreNodes: []string{
					node2.Name,
				},
			},
			wantPodByNode: map[string]*corev1.Pod{
				"node1": pod1Node1,
				"node3": nil,
			},
			wantPodToDelete:     nil,
			wantUnscheduledPods: nil,
		},
		{
			name: "ignore node2",
			args: args{
				replicaset: datadoghqv1alpha1test.NewExtendedDaemonSetReplicaSet("foo", "bar", nil),
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node1, *node2, *node3},
				},
				podList: &corev1.PodList{
					Items: []corev1.Pod{},
				},
				ignoreNodes: []string{"node2"},
			},
			wantPodByNode: map[string]*corev1.Pod{
				"node1": nil,
				"node3": nil,
			},
			wantPodToDelete:     nil,
			wantUnscheduledPods: nil,
		},

		{
			name: "ignore node2 + 3 pods",
			args: args{
				replicaset: datadoghqv1alpha1test.NewExtendedDaemonSetReplicaSet("foo", "bar", nil),
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node1, *node2, *node3},
				},
				podList: &corev1.PodList{
					Items: []corev1.Pod{
						*pod1Node1,
						*pod2Node2,
						*pod3Node3,
					},
				},
				ignoreNodes: []string{},
			},
			wantPodByNode: map[string]*corev1.Pod{
				"node1": pod1Node1,
				"node2": pod2Node2,
				"node3": pod3Node3,
			},
			wantPodToDelete:     nil,
			wantUnscheduledPods: nil,
		},
		{
			name: "pod deletion support",
			args: args{
				replicaset: datadoghqv1alpha1test.NewExtendedDaemonSetReplicaSet("foo", "bar", nil),
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node3},
				},
				podList: &corev1.PodList{
					Items: []corev1.Pod{
						*pod3NodeFake,
					},
				},
				ignoreNodes: []string{},
			},
			wantPodByNode: map[string]*corev1.Pod{
				"node3": nil,
			},
			wantPodToDelete:     []*corev1.Pod{pod3NodeFake},
			wantUnscheduledPods: nil,
		},
		{
			name: "pod deletion support, already in deletion state",
			args: args{
				replicaset: datadoghqv1alpha1test.NewExtendedDaemonSetReplicaSet("foo", "bar", nil),
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node3},
				},
				podList: &corev1.PodList{
					Items: []corev1.Pod{
						*pod3NodeFakeBis,
					},
				},
				ignoreNodes: []string{},
			},
			wantPodByNode: map[string]*corev1.Pod{
				"node3": nil,
			},
			wantPodToDelete:     nil,
			wantUnscheduledPods: nil,
		},
		{
			name: "filter pod unknow status phase",
			args: args{
				replicaset: datadoghqv1alpha1test.NewExtendedDaemonSetReplicaSet("foo", "bar", nil),
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node1},
				},
				podList: &corev1.PodList{
					Items: []corev1.Pod{
						*pod1Node1,
						*pod4Node1,
					},
				},
				ignoreNodes: []string{},
			},
			wantPodByNode: map[string]*corev1.Pod{
				"node1": pod1Node1,
			},
			wantPodToDelete:     nil,
			wantUnscheduledPods: nil,
		},
		{
			name: "don't filter node not ready unknow status phase",
			args: args{
				replicaset: datadoghqv1alpha1test.NewExtendedDaemonSetReplicaSet("foo", "bar", nil),
				nodeList: &corev1.NodeList{
					Items: []corev1.Node{*node1, *node4},
				},
				podList: &corev1.PodList{
					Items: []corev1.Pod{
						*pod1Node1,
						*pod4Node1,
					},
				},
				ignoreNodes: []string{},
			},
			wantPodByNode: map[string]*corev1.Pod{
				"node1": pod1Node1,
				"node4": nil,
			},
			wantPodToDelete:     nil,
			wantUnscheduledPods: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqLogger := log.WithValues("test:", tt.name)
			gotPodByNode, gotPodToDelete, gotUnscheduledPods := FilterAndMapPodsByNode(reqLogger, tt.args.replicaset, tt.args.nodeList, tt.args.podList, tt.args.ignoreNodes)
			if !apiequality.Semantic.DeepEqual(gotPodByNode, tt.wantPodByNode) {
				t.Errorf("FilterAndMapPodsByNode() gotPodByNode = %v, want %v", gotPodByNode, tt.wantPodByNode)
			}
			if !apiequality.Semantic.DeepEqual(gotPodToDelete, tt.wantPodToDelete) {
				t.Errorf("FilterAndMapPodsByNode() gotPodToDelete = %#v, want %#v", gotPodToDelete, tt.wantPodToDelete)
			}
			if !apiequality.Semantic.DeepEqual(gotUnscheduledPods, tt.wantUnscheduledPods) {
				t.Errorf("FilterAndMapPodsByNode() gotUnscheduledPods = %#v, want %#v", gotUnscheduledPods, tt.wantUnscheduledPods)
			}
		})
	}
}
