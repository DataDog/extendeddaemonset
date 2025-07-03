// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package strategy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/pkg/controller/test"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManageUnknown(t *testing.T) {
	logf.SetLogger(zap.New())
	logger := logf.Log.WithName("TestManageUnknown")

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = datadoghqv1alpha1.AddToScheme(scheme)

	// Create test nodes
	node1 := test.NewNode("node1", &test.NewNodeOptions{})
	node2 := test.NewNode("node2", &test.NewNodeOptions{})

	// Create test pods with matching template hash
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "test-ns",
			Annotations: map[string]string{
				datadoghqv1alpha1.MD5ExtendedDaemonSetAnnotationKey: "test-hash",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: node1.Name,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "test-ns",
			Annotations: map[string]string{
				datadoghqv1alpha1.MD5ExtendedDaemonSetAnnotationKey: "test-hash",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: node2.Name,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	// Create test ERS
	ers := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ers",
			Namespace: "test-ns",
		},
		Spec: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetSpec{
			TemplateGeneration: "test-hash",
		},
		Status: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus{},
	}

	tests := []struct {
		name           string
		podByNodeName  map[*NodeItem]*corev1.Pod
		podToCleanUp   []*corev1.Pod
		expectedResult *Result
		expectError    bool
	}{
		{
			name: "no pods",
			podByNodeName: map[*NodeItem]*corev1.Pod{
				NewNodeItem(node1, nil): nil,
				NewNodeItem(node2, nil): nil,
			},
			podToCleanUp: []*corev1.Pod{},
			expectedResult: &Result{
				NewStatus: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus{
					Status:                   "unknown",
					Desired:                  0,
					Ready:                    0,
					Current:                  0,
					Available:                0,
					IgnoredUnresponsiveNodes: 0,
				},
			},
			expectError: false,
		},
		{
			name: "pods present, no cleanup needed",
			podByNodeName: map[*NodeItem]*corev1.Pod{
				NewNodeItem(node1, nil): pod1,
				NewNodeItem(node2, nil): pod2,
			},
			podToCleanUp: []*corev1.Pod{},
			expectedResult: &Result{
				NewStatus: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus{
					Status:                   "unknown",
					Desired:                  0,
					Ready:                    2,
					Current:                  2,
					Available:                2,
					IgnoredUnresponsiveNodes: 0,
				},
			},
			expectError: false,
		},
		{
			name: "pods need cleanup",
			podByNodeName: map[*NodeItem]*corev1.Pod{
				NewNodeItem(node1, nil): pod1,
				NewNodeItem(node2, nil): pod2,
			},
			podToCleanUp: []*corev1.Pod{pod1, pod2},
			expectedResult: &Result{
				NewStatus: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus{
					Status:                   "unknown",
					Desired:                  0,
					Ready:                    2,
					Current:                  2,
					Available:                2,
					IgnoredUnresponsiveNodes: 0,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with the pods that need to be cleaned up
			clientObjects := []runtime.Object{}
			for _, pod := range tt.podToCleanUp {
				clientObjects = append(clientObjects, pod)
			}
			client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(clientObjects...).Build()

			params := &Parameters{
				Replicaset:    ers,
				Logger:        logger,
				NewStatus:     &ers.Status,
				NodeByName:    map[string]*NodeItem{},
				PodByNodeName: tt.podByNodeName,
				PodToCleanUp:  tt.podToCleanUp,
				CanaryNodes:   []string{},
			}

			// Add nodes to NodeByName map
			for nodeItem := range tt.podByNodeName {
				params.NodeByName[nodeItem.Node.Name] = nodeItem
			}

			result, err := ManageUnknown(client, params)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			require.NotNil(t, result)
			require.NotNil(t, result.NewStatus)

			assert.Equal(t, tt.expectedResult.NewStatus.Status, result.NewStatus.Status)
			assert.Equal(t, tt.expectedResult.NewStatus.Desired, result.NewStatus.Desired)
			assert.Equal(t, tt.expectedResult.NewStatus.Ready, result.NewStatus.Ready)
			assert.Equal(t, tt.expectedResult.NewStatus.Current, result.NewStatus.Current)
			assert.Equal(t, tt.expectedResult.NewStatus.Available, result.NewStatus.Available)
			assert.Equal(t, tt.expectedResult.NewStatus.IgnoredUnresponsiveNodes, result.NewStatus.IgnoredUnresponsiveNodes)

			// Verify that requeue is set correctly
			if result.NewStatus.Desired != result.NewStatus.Ready {
				assert.True(t, result.Result.Requeue)
			}
			assert.Equal(t, time.Second, result.Result.RequeueAfter)
		})
	}
}

func TestManageUnknown_WithCanaryNodes(t *testing.T) {
	logf.SetLogger(zap.New())
	logger := logf.Log.WithName("test")

	// Create test nodes
	node1 := test.NewNode("node1", nil)
	node2 := test.NewNode("canary-node", nil)

	// Create test pods with matching template hash
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "test",
			Annotations: map[string]string{
				datadoghqv1alpha1.MD5ExtendedDaemonSetAnnotationKey: "test-hash",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	canaryPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "canary-pod",
			Namespace: "test",
			Annotations: map[string]string{
				datadoghqv1alpha1.MD5ExtendedDaemonSetAnnotationKey: "test-hash",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "canary-node",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	// Create fake client
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod1, canaryPod).Build()

	// Create test ERS
	ers := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ers",
			Namespace: "test",
		},
		Spec: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetSpec{
			TemplateGeneration: "test-hash",
		},
		Status: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus{
			Status: "unknown",
		},
	}

	nodeItem1 := NewNodeItem(node1, nil)
	canaryNodeItem := NewNodeItem(node2, nil)

	params := &Parameters{
		Replicaset: ers,
		Logger:     logger,
		NewStatus:  &ers.Status,
		NodeByName: map[string]*NodeItem{
			"node1":       nodeItem1,
			"canary-node": canaryNodeItem,
		},
		PodByNodeName: map[*NodeItem]*corev1.Pod{
			nodeItem1:      pod1,
			canaryNodeItem: canaryPod,
		},
		PodToCleanUp: []*corev1.Pod{},
		CanaryNodes:  []string{"canary-node"}, // This node should be excluded
	}

	result, err := ManageUnknown(client, params)

	assert.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.NewStatus)

	// Verify that only non-canary pods are counted
	assert.Equal(t, "unknown", result.NewStatus.Status)
	assert.Equal(t, int32(0), result.NewStatus.Desired)
	assert.Equal(t, int32(1), result.NewStatus.Ready)     // Only pod1, not canaryPod
	assert.Equal(t, int32(1), result.NewStatus.Current)   // Only pod1, not canaryPod
	assert.Equal(t, int32(1), result.NewStatus.Available) // Only pod1, not canaryPod
}
