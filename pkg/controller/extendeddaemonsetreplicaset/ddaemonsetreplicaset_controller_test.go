// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

import (
	"reflect"
	"testing"
	"time"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1/test"
	"github.com/datadog/extendeddaemonset/pkg/controller/extendeddaemonsetreplicaset/strategy"
	ctrltest "github.com/datadog/extendeddaemonset/pkg/controller/test"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestReconcileExtendedDaemonSetReplicaSet_Reconcile(t *testing.T) {
	eventBroadcaster := record.NewBroadcaster()
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "TestReconcileExtendedDaemonSet_Reconcile"})
	logf.SetLogger(logf.ZapLogger(true))

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(datadoghqv1alpha1.SchemeGroupVersion, &datadoghqv1alpha1.ExtendedDaemonSetReplicaSetList{})
	s.AddKnownTypes(datadoghqv1alpha1.SchemeGroupVersion, &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{})
	s.AddKnownTypes(datadoghqv1alpha1.SchemeGroupVersion, &datadoghqv1alpha1.ExtendedDaemonSetList{})
	s.AddKnownTypes(datadoghqv1alpha1.SchemeGroupVersion, &datadoghqv1alpha1.ExtendedDaemonSet{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.PodList{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Pod{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.NodeList{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Node{})

	daemonset := test.NewExtendedDaemonSet("but", "foo", &test.NewExtendedDaemonSetOptions{Labels: map[string]string{"foo-key": "bar-value"}})

	status := &datadoghqv1alpha1.ExtendedDaemonSetStatus{
		ActiveReplicaSet: "foo-1",
		Canary: &datadoghqv1alpha1.ExtendedDaemonSetStatusCanary{
			ReplicaSet: "foo-2",
		},
	}
	daemonsetWithStatus := test.NewExtendedDaemonSet("but", "foo", &test.NewExtendedDaemonSetOptions{Labels: map[string]string{"foo-key": "bar-value"}, Status: status})
	maxUnavailable := intstr.FromInt(1)
	slowStartAdditiveIncrease := intstr.FromInt(1)
	slowStartIntervalDuration := metav1.Duration{Duration: time.Minute}
	replicaset := test.NewExtendedDaemonSetReplicaSet("but", "foo-1", &test.NewExtendedDaemonSetReplicaSetOptions{
		Labels:       map[string]string{"foo-key": "bar-value"},
		OwnerRefName: "foo",
		RollingUpdate: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
			MaxUnavailable:            &maxUnavailable,
			SlowStartAdditiveIncrease: &slowStartAdditiveIncrease,
			SlowStartIntervalDuration: &slowStartIntervalDuration,
			MaxParallelPodCreation:    datadoghqv1alpha1.NewInt32(1),
		},
	})

	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		recorder record.EventRecorder
	}
	type args struct {
		request reconcile.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    reconcile.Result
		wantErr bool
	}{
		{
			name: "ReplicaSet does not exist in client",
			fields: fields{
				client:   fake.NewFakeClient(),
				scheme:   s,
				recorder: recorder,
			},
			args: args{
				request: newRequest("bar", "foo-bar"),
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name: "ReplicaSet exist but not Daemonset, it should trigger an error",
			fields: fields{
				client:   fake.NewFakeClient(replicaset),
				scheme:   s,
				recorder: recorder,
			},
			args: args{
				request: newRequest("but", "foo-1"),
			},
			want:    reconcile.Result{},
			wantErr: true,
		},
		{
			name: "ReplicaSet, Daemonset exist, but no status Daemonset => should requeue in 1sec",
			fields: fields{
				client:   fake.NewFakeClient(daemonset, replicaset),
				scheme:   s,
				recorder: recorder,
			},
			args: args{
				request: newRequest("but", "foo-1"),
			},
			want:    reconcile.Result{RequeueAfter: time.Second},
			wantErr: false,
		},
		{
			name: "ReplicaSet, Daemonset exist with status",
			fields: fields{
				client:   fake.NewFakeClient(daemonsetWithStatus, replicaset),
				scheme:   s,
				recorder: recorder,
			},
			args: args{
				request: newRequest("but", "foo-1"),
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileExtendedDaemonSetReplicaSet{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				recorder: tt.fields.recorder,
			}
			got, err := r.Reconcile(tt.args.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_retrieveReplicaSetStatus(t *testing.T) {
	status := &datadoghqv1alpha1.ExtendedDaemonSetStatus{
		ActiveReplicaSet: "rs-active",
		Canary: &datadoghqv1alpha1.ExtendedDaemonSetStatusCanary{
			ReplicaSet: "rs-canary",
		},
	}
	daemonset := test.NewExtendedDaemonSet("bar", "foo", &test.NewExtendedDaemonSetOptions{Labels: map[string]string{"foo-key": "bar-value"}, Status: status})

	type args struct {
		daemonset       *datadoghqv1alpha1.ExtendedDaemonSet
		replicassetName string
	}
	tests := []struct {
		name string
		args args
		want strategy.ReplicaSetStatus
	}{
		{
			name: "status unknow",
			args: args{
				daemonset:       daemonset,
				replicassetName: "rs-unknow",
			},
			want: strategy.ReplicaSetStatusUnknown,
		},
		{
			name: "status unknow",
			args: args{
				daemonset:       daemonset,
				replicassetName: "rs-active",
			},
			want: strategy.ReplicaSetStatusActive,
		},
		{
			name: "status unknow",
			args: args{
				daemonset:       daemonset,
				replicassetName: "rs-canary",
			},
			want: strategy.ReplicaSetStatusCanary,
		},
		{
			name: "activeRS not set => unknow",
			args: args{
				daemonset:       test.NewExtendedDaemonSet("bar", "foo", &test.NewExtendedDaemonSetOptions{Labels: map[string]string{"foo-key": "bar-value"}}),
				replicassetName: "rs-active",
			},
			want: strategy.ReplicaSetStatusUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retrieveReplicaSetStatus(tt.args.daemonset, tt.args.replicassetName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("retrieveReplicaSetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_retrieveOwnerReference(t *testing.T) {
	type args struct {
		obj *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "ownerref not found",
			args: args{
				obj: test.NewExtendedDaemonSetReplicaSet("bar", "foo-1", &test.NewExtendedDaemonSetReplicaSetOptions{
					Labels: map[string]string{"foo-key": "bar-value"},
				}),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "ownerref found",
			args: args{
				obj: test.NewExtendedDaemonSetReplicaSet("bar", "foo-1", &test.NewExtendedDaemonSetReplicaSetOptions{
					Labels:       map[string]string{"foo-key": "bar-value"},
					OwnerRefName: "foo",
				}),
			},
			want:    "foo",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := retrieveOwnerReference(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("retrieveOwnerReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("retrieveOwnerReference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileExtendedDaemonSetReplicaSet_getPodList(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(datadoghqv1alpha1.SchemeGroupVersion, &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Pod{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.PodList{})

	ns := "bar"
	daemonset := test.NewExtendedDaemonSet(ns, "foo", &test.NewExtendedDaemonSetOptions{Labels: map[string]string{"foo-key": "bar-value"}})
	podOptions := &ctrltest.NewPodOptions{
		Labels: map[string]string{
			datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey: "foo",
		},
	}
	pod1 := ctrltest.NewPod(ns, "foo-pod1", ns, podOptions)
	pod2 := ctrltest.NewPod(ns, "foo-pod2", ns, podOptions)

	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		recorder record.EventRecorder
	}
	type args struct {
		ds *datadoghqv1alpha1.ExtendedDaemonSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *corev1.PodList
		wantErr bool
	}{
		{
			name: "no pods",
			fields: fields{
				client: fake.NewFakeClient(),
				scheme: s,
			},
			args: args{
				ds: daemonset,
			},
			want:    &corev1.PodList{},
			wantErr: false,
		},
		{
			name: "two pods",
			fields: fields{
				client: fake.NewFakeClient(pod1, pod2),
				scheme: s,
			},
			args: args{
				ds: daemonset,
			},
			want: &corev1.PodList{
				Items: []corev1.Pod{*pod1, *pod2},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileExtendedDaemonSetReplicaSet{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				recorder: tt.fields.recorder,
			}
			got, err := r.getPodList(tt.args.ds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.getPodList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.getPodList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileExtendedDaemonSetReplicaSet_getNodeList(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Node{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.NodeList{})

	replicasset := test.NewExtendedDaemonSetReplicaSet("bar", "foo-1", &test.NewExtendedDaemonSetReplicaSetOptions{
		Labels: map[string]string{"foo-key": "bar-value"}})

	nodeOptions := &ctrltest.NewNodeOptions{
		Conditions: []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			},
		},
	}
	node1 := ctrltest.NewNode("node1", nodeOptions)
	node2 := ctrltest.NewNode("node2", nodeOptions)

	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		recorder record.EventRecorder
	}
	type args struct {
		replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *corev1.NodeList
		wantErr bool
	}{
		{
			name: "no nodes",
			fields: fields{
				client: fake.NewFakeClient(node1, node2),
				scheme: s,
			},
			args: args{
				replicaset: replicasset,
			},
			want: &corev1.NodeList{
				Items: []corev1.Node{*node1, *node2},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileExtendedDaemonSetReplicaSet{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				recorder: tt.fields.recorder,
			}
			got, err := r.getNodeList(tt.args.replicaset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.getNodeList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.getNodeList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileExtendedDaemonSetReplicaSet_getDaemonsetOwner(t *testing.T) {

	s := scheme.Scheme
	s.AddKnownTypes(datadoghqv1alpha1.SchemeGroupVersion, &datadoghqv1alpha1.ExtendedDaemonSet{})

	replicasset := test.NewExtendedDaemonSetReplicaSet("bar", "foo-1", &test.NewExtendedDaemonSetReplicaSetOptions{
		Labels: map[string]string{"foo-key": "bar-value"}})
	replicassetWithOwner := test.NewExtendedDaemonSetReplicaSet("bar", "foo-1", &test.NewExtendedDaemonSetReplicaSetOptions{
		Labels:       map[string]string{"foo-key": "bar-value"},
		OwnerRefName: "foo"},
	)
	daemonset := test.NewExtendedDaemonSet("bar", "foo", &test.NewExtendedDaemonSetOptions{Labels: map[string]string{"foo-key": "bar-value"}})

	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		recorder record.EventRecorder
	}
	type args struct {
		replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *datadoghqv1alpha1.ExtendedDaemonSet
		wantErr bool
	}{
		{
			name: "owner not define, return errror",
			fields: fields{
				client: fake.NewFakeClient(),
				scheme: s,
			},
			args: args{
				replicaset: replicasset,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "with owner define, but not exist, return errror",
			fields: fields{
				client: fake.NewFakeClient(),
				scheme: s,
			},
			args: args{
				replicaset: replicassetWithOwner,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "with owner define, but not exist, return errror",
			fields: fields{
				client: fake.NewFakeClient(daemonset),
				scheme: s,
			},
			args: args{
				replicaset: replicassetWithOwner,
			},
			want:    daemonset,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileExtendedDaemonSetReplicaSet{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				recorder: tt.fields.recorder,
			}
			got, err := r.getDaemonsetOwner(tt.args.replicaset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.getDaemonsetOwner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !apiequality.Semantic.DeepEqual(got, tt.want) {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.getDaemonsetOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileExtendedDaemonSetReplicaSet_updateReplicaSet(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(datadoghqv1alpha1.SchemeGroupVersion, &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{})

	replicasset := test.NewExtendedDaemonSetReplicaSet("bar", "foo-1", &test.NewExtendedDaemonSetReplicaSetOptions{
		Labels:       map[string]string{"foo-key": "bar-value"},
		OwnerRefName: "foo"},
	)

	newStatus := replicasset.Status.DeepCopy()
	newStatus.Desired = 3

	type fields struct {
		client   client.Client
		scheme   *runtime.Scheme
		recorder record.EventRecorder
	}
	type args struct {
		replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
		newStatus  *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error: replicaset doesn't exist",
			fields: fields{
				client: fake.NewFakeClient(),
				scheme: s,
			},
			args: args{
				replicaset: replicasset,
				newStatus:  newStatus,
			},
			wantErr: true,
		},
		{
			name: "new status, update should work",
			fields: fields{
				client: fake.NewFakeClient(replicasset),
				scheme: s,
			},
			args: args{
				replicaset: replicasset,
				newStatus:  newStatus,
			},
			wantErr: false,
		},
		{
			name: "same status, we should not update the replicaset",
			fields: fields{
				client: fake.NewFakeClient(),
				scheme: s,
			},
			args: args{
				replicaset: replicasset,
				newStatus:  replicasset.Status.DeepCopy(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ReconcileExtendedDaemonSetReplicaSet{
				client:   tt.fields.client,
				scheme:   tt.fields.scheme,
				recorder: tt.fields.recorder,
			}
			if err := r.updateReplicaSet(tt.args.replicaset, tt.args.newStatus); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileExtendedDaemonSetReplicaSet.updateReplicaSet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newRequest(ns, name string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: ns,
			Name:      name,
		},
	}
}
