// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package strategy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
)

var (
	testLogger          = logf.Log.WithName("test")
	testCanaryNodeNames = []string{"a", "b", "c"}
	testCanaryNodes     = map[string]*NodeItem{
		"a": {
			Node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "a",
				},
			},
		},
		"b": {
			Node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "b",
				},
			},
		},
		"c": {
			Node: &v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "c",
				},
			},
		},
	}
	readyPodStatus = v1.PodStatus{
		Conditions: []v1.PodCondition{
			{
				Type:   v1.PodReady,
				Status: v1.ConditionTrue,
			},
		},
	}
)

func newTestCanaryPod(name, hash string, status v1.PodStatus) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				"extendeddaemonset.datadoghq.com/templatehash": hash,
			},
		},
		Status: status,
	}
}

func podTerminatedStatus(restartCount int32, reason string, time time.Time) v1.PodStatus {
	return v1.PodStatus{
		ContainerStatuses: []v1.ContainerStatus{
			{
				Name:         "restarting",
				RestartCount: restartCount,
				LastTerminationState: v1.ContainerState{
					Terminated: &v1.ContainerStateTerminated{
						Reason:     reason,
						FinishedAt: metav1.NewTime(time),
					},
				},
			},
		},
	}
}

func podWaitingStatus(reason, message string) v1.PodStatus {
	return v1.PodStatus{
		ContainerStatuses: []v1.ContainerStatus{
			{
				Name: "waiting",
				State: v1.ContainerState{
					Waiting: &v1.ContainerStateWaiting{

						Reason:  reason,
						Message: message,
					},
				},
			},
		},
	}
}

func withDeletionTimestamp(pod *v1.Pod) *v1.Pod {
	ts := metav1.Now()
	pod.DeletionTimestamp = &ts

	return pod
}

func withHostIP(pod *v1.Pod, ip string) *v1.Pod {
	pod.Status.HostIP = ip

	return pod
}

type canaryStatusTest struct {
	annotations map[string]string
	params      *Parameters
	result      *Result
	now         time.Time
}

func (test *canaryStatusTest) Run(t *testing.T) {
	if test.now.IsZero() {
		test.now = time.Now()
	}
	result := manageCanaryStatus(test.annotations, test.params, test.now)
	assert.Equal(t, test.result, result)
}

func TestManageCanaryStatus_NoRestartsAndPodsToCreate(t *testing.T) {
	test := canaryStatusTest{
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(5),
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus:   &v1alpha1.ExtendedDaemonSetReplicaSetStatus{},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: newTestCanaryPod("foo-a", "v1", readyPodStatus),
				testCanaryNodes["b"]: nil,
				testCanaryNodes["c"]: nil,
			},
			Logger: testLogger,
		},
		result: &Result{
			PodsToCreate: []*NodeItem{
				testCanaryNodes["b"],
				testCanaryNodes["c"],
			},
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary",
				Desired:   3,
				Current:   1,
				Ready:     1,
				Available: 1,
			},
			Result: requeuePromptly(),
		},
	}
	test.Run(t)
}

func TestManageCanaryStatus_NoRestartsAndNoPodsToCreate(t *testing.T) {
	test := canaryStatusTest{
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(3),
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus:   &v1alpha1.ExtendedDaemonSetReplicaSetStatus{},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: newTestCanaryPod("foo-a", "v1", readyPodStatus),
				testCanaryNodes["b"]: newTestCanaryPod("foo-b", "v1", readyPodStatus),
				testCanaryNodes["c"]: newTestCanaryPod("foo-c", "v1", readyPodStatus),
			},
			Logger: testLogger,
		},
		result: &Result{
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary",
				Desired:   3,
				Current:   3,
				Ready:     3,
				Available: 3,
			},
			Result: reconcile.Result{},
		},
	}
	test.Run(t)
}

func TestManageCanaryStatus_NoRestartsAndPodWithDeletionTimestamp(t *testing.T) {
	test := canaryStatusTest{
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(5),
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus:   &v1alpha1.ExtendedDaemonSetReplicaSetStatus{},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: withHostIP(withDeletionTimestamp(newTestCanaryPod("foo-a", "v1", readyPodStatus)), "1.2.3.4"),
				testCanaryNodes["b"]: nil,
				testCanaryNodes["c"]: nil,
			},
			Logger: testLogger,
		},
		result: &Result{
			PodsToCreate: []*NodeItem{
				testCanaryNodes["b"],
				testCanaryNodes["c"],
			},
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary",
				Desired:   3,
				Current:   0,
				Ready:     0,
				Available: 0,
			},
			Result: requeuePromptly(),
		},
	}
	test.Run(t)
}

func TestManageCanaryStatus_NoRestartsAndPodsToDelete(t *testing.T) {
	test := canaryStatusTest{
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(5),
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus:   &v1alpha1.ExtendedDaemonSetReplicaSetStatus{},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: newTestCanaryPod("foo-a", "v0", readyPodStatus),
				testCanaryNodes["b"]: nil,
				testCanaryNodes["c"]: nil,
			},
			Logger: testLogger,
		},
		result: &Result{
			PodsToCreate: []*NodeItem{
				testCanaryNodes["b"],
				testCanaryNodes["c"],
			},
			PodsToDelete: []*NodeItem{
				testCanaryNodes["a"],
			},
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary",
				Desired:   3,
				Current:   0,
				Ready:     0,
				Available: 0,
			},
			Result: requeuePromptly(),
		},
	}
	test.Run(t)
}

func TestManageCanaryStatus_HighRestartsLeadingToPause(t *testing.T) {
	now := time.Now()
	restartedAt := now.Add(-time.Minute)
	test := canaryStatusTest{
		now: now,
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(5),
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus:   &v1alpha1.ExtendedDaemonSetReplicaSetStatus{},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: newTestCanaryPod("foo-a", "v1", podTerminatedStatus(3, "CrashLoopBackOff", restartedAt)),
				testCanaryNodes["b"]: nil,
				testCanaryNodes["c"]: nil,
			},
			Logger: testLogger,
		},
		result: &Result{
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary",
				Desired:   3,
				Current:   1,
				Ready:     0,
				Available: 0,
				Conditions: []v1alpha1.ExtendedDaemonSetReplicaSetCondition{
					{
						Type:               v1alpha1.ConditionTypeCanaryPaused,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(now),
						LastUpdateTime:     metav1.NewTime(now),
						Reason:             "CrashLoopBackOff",
						Message:            "",
					},
					{
						Type:               v1alpha1.ConditionTypePodRestarting,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(restartedAt),
						LastUpdateTime:     metav1.NewTime(restartedAt),
						Message:            "Pod foo-a restarting with reason: CrashLoopBackOff",
					},
				},
			},
			IsPaused:     true,
			PausedReason: v1alpha1.ExtendedDaemonSetStatusReasonCLB,
			Result:       reconcile.Result{},
		},
	}
	test.Run(t)
}

func TestManageCanaryStatus_HighRestartsLeadingToFail(t *testing.T) {
	now := time.Now()
	restartedAt := now.Add(-time.Minute)
	test := canaryStatusTest{
		now: now,
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(5),
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus:   &v1alpha1.ExtendedDaemonSetReplicaSetStatus{},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: newTestCanaryPod("foo-a", "v1", podTerminatedStatus(6, "CrashLoopBackOff", restartedAt)),
				testCanaryNodes["b"]: nil,
				testCanaryNodes["c"]: nil,
			},
			Logger: testLogger,
		},
		result: &Result{
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary-failed",
				Desired:   3,
				Current:   1,
				Ready:     0,
				Available: 0,
				Conditions: []v1alpha1.ExtendedDaemonSetReplicaSetCondition{
					{
						Type:               v1alpha1.ConditionTypeCanaryFailed,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(now),
						LastUpdateTime:     metav1.NewTime(now),
						Reason:             "CrashLoopBackOff",
						Message:            "",
					},
					{
						Type:               v1alpha1.ConditionTypePodRestarting,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(restartedAt),
						LastUpdateTime:     metav1.NewTime(restartedAt),
						Message:            "Pod foo-a restarting with reason: CrashLoopBackOff",
					},
				},
			},
			IsFailed:     true,
			FailedReason: v1alpha1.ExtendedDaemonSetStatusReasonCLB,
			Result:       reconcile.Result{},
		},
	}
	test.Run(t)
}

func TestManageCanaryStatus_LongRestartsDurationLeadingToFail(t *testing.T) {
	now := time.Now()
	restartsStartedAt := now.Add(-time.Hour)
	restartsUpdatedAt := now.Add(-10 * time.Minute)
	restartedAt := now.Add(-time.Minute)

	test := canaryStatusTest{
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:             v1alpha1.NewBool(true),
						MaxRestarts:         v1alpha1.NewInt32(5),
						MaxRestartsDuration: &metav1.Duration{Duration: 20 * time.Minute},
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Conditions: []v1alpha1.ExtendedDaemonSetReplicaSetCondition{
					{
						Type:               v1alpha1.ConditionTypePodRestarting,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(restartsStartedAt),
						LastUpdateTime:     metav1.NewTime(restartsUpdatedAt),
						Message:            "Pod foo-b restarting with reason: CrashLoopBackOff",
					},
				},
			},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: newTestCanaryPod("foo-a", "v1", podTerminatedStatus(1, "CrashLoopBackOff", restartedAt)),
				testCanaryNodes["b"]: nil,
				testCanaryNodes["c"]: nil,
			},
			Logger: testLogger,
		},
		result: &Result{
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary-failed",
				Desired:   3,
				Current:   1,
				Ready:     0,
				Available: 0,
				Conditions: []v1alpha1.ExtendedDaemonSetReplicaSetCondition{
					{
						Type:               v1alpha1.ConditionTypePodRestarting,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(restartsStartedAt),
						LastUpdateTime:     metav1.NewTime(restartedAt),
						Message:            "Pod foo-a restarting with reason: CrashLoopBackOff",
					},
				},
			},
			IsFailed:     true,
			FailedReason: v1alpha1.ExtendedDaemonSetStatusRestartsTimeoutExceeded,
			Result:       reconcile.Result{},
		},
	}
	test.Run(t)
}

func TestManageCanaryStatus_ImagePullErrorLeadingToPause(t *testing.T) {
	now := time.Now()
	test := canaryStatusTest{
		params: &Parameters{
			EDSName: "foo",
			Strategy: &v1alpha1.ExtendedDaemonSetSpecStrategy{
				Canary: &v1alpha1.ExtendedDaemonSetSpecStrategyCanary{
					AutoPause: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoPause{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(2),
					},
					AutoFail: &v1alpha1.ExtendedDaemonSetSpecStrategyCanaryAutoFail{
						Enabled:     v1alpha1.NewBool(true),
						MaxRestarts: v1alpha1.NewInt32(5),
					},
				},
			},
			Replicaset: &v1alpha1.ExtendedDaemonSetReplicaSet{
				Spec: v1alpha1.ExtendedDaemonSetReplicaSetSpec{
					TemplateGeneration: "v1",
				},
			},
			NewStatus:   &v1alpha1.ExtendedDaemonSetReplicaSetStatus{},
			CanaryNodes: testCanaryNodeNames,
			NodeByName:  testCanaryNodes,
			PodByNodeName: map[*NodeItem]*v1.Pod{
				testCanaryNodes["a"]: newTestCanaryPod("foo-a", "v1", podWaitingStatus("ImagePullBackOff", `Back-off pulling image "gcr.io/missing"`)),
				testCanaryNodes["b"]: nil,
				testCanaryNodes["c"]: nil,
			},
			Logger: testLogger,
		},
		result: &Result{
			NewStatus: &v1alpha1.ExtendedDaemonSetReplicaSetStatus{
				Status:    "canary",
				Desired:   3,
				Current:   1,
				Ready:     0,
				Available: 0,
				Conditions: []v1alpha1.ExtendedDaemonSetReplicaSetCondition{
					{
						Type:               v1alpha1.ConditionTypeCanaryPaused,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(now),
						LastUpdateTime:     metav1.NewTime(now),
						Reason:             "ImagePullBackOff",
					},
					{
						Type:               v1alpha1.ConditionTypePodCannotStart,
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(now),
						LastUpdateTime:     metav1.NewTime(now),
						Reason:             "ImagePullBackOff",
						Message:            "Pod foo-a cannot start with reason: ImagePullBackOff",
					},
				},
			},
			IsPaused:     true,
			PausedReason: v1alpha1.ExtendedDaemonSetStatusReason("ImagePullBackOff"),
			Result:       reconcile.Result{},
		},
		now: now,
	}
	test.Run(t)
}
