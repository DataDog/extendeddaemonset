//go:build !ignore_autogenerated

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSet) DeepCopyInto(out *ExtendedDaemonSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSet.
func (in *ExtendedDaemonSet) DeepCopy() *ExtendedDaemonSet {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExtendedDaemonSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetCondition) DeepCopyInto(out *ExtendedDaemonSetCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetCondition.
func (in *ExtendedDaemonSetCondition) DeepCopy() *ExtendedDaemonSetCondition {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetList) DeepCopyInto(out *ExtendedDaemonSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExtendedDaemonSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetList.
func (in *ExtendedDaemonSetList) DeepCopy() *ExtendedDaemonSetList {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExtendedDaemonSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetReplicaSet) DeepCopyInto(out *ExtendedDaemonSetReplicaSet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetReplicaSet.
func (in *ExtendedDaemonSetReplicaSet) DeepCopy() *ExtendedDaemonSetReplicaSet {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetReplicaSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExtendedDaemonSetReplicaSet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetReplicaSetCondition) DeepCopyInto(out *ExtendedDaemonSetReplicaSetCondition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetReplicaSetCondition.
func (in *ExtendedDaemonSetReplicaSetCondition) DeepCopy() *ExtendedDaemonSetReplicaSetCondition {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetReplicaSetCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetReplicaSetList) DeepCopyInto(out *ExtendedDaemonSetReplicaSetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExtendedDaemonSetReplicaSet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetReplicaSetList.
func (in *ExtendedDaemonSetReplicaSetList) DeepCopy() *ExtendedDaemonSetReplicaSetList {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetReplicaSetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExtendedDaemonSetReplicaSetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetReplicaSetSpec) DeepCopyInto(out *ExtendedDaemonSetReplicaSetSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	in.Template.DeepCopyInto(&out.Template)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetReplicaSetSpec.
func (in *ExtendedDaemonSetReplicaSetSpec) DeepCopy() *ExtendedDaemonSetReplicaSetSpec {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetReplicaSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetReplicaSetSpecStrategy) DeepCopyInto(out *ExtendedDaemonSetReplicaSetSpecStrategy) {
	*out = *in
	in.RollingUpdate.DeepCopyInto(&out.RollingUpdate)
	out.ReconcileFrequency = in.ReconcileFrequency
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetReplicaSetSpecStrategy.
func (in *ExtendedDaemonSetReplicaSetSpecStrategy) DeepCopy() *ExtendedDaemonSetReplicaSetSpecStrategy {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetReplicaSetSpecStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetReplicaSetStatus) DeepCopyInto(out *ExtendedDaemonSetReplicaSetStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ExtendedDaemonSetReplicaSetCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetReplicaSetStatus.
func (in *ExtendedDaemonSetReplicaSetStatus) DeepCopy() *ExtendedDaemonSetReplicaSetStatus {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetReplicaSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetSpec) DeepCopyInto(out *ExtendedDaemonSetSpec) {
	*out = *in
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	in.Template.DeepCopyInto(&out.Template)
	in.Strategy.DeepCopyInto(&out.Strategy)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetSpec.
func (in *ExtendedDaemonSetSpec) DeepCopy() *ExtendedDaemonSetSpec {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetSpecStrategy) DeepCopyInto(out *ExtendedDaemonSetSpecStrategy) {
	*out = *in
	in.RollingUpdate.DeepCopyInto(&out.RollingUpdate)
	if in.Canary != nil {
		in, out := &in.Canary, &out.Canary
		*out = new(ExtendedDaemonSetSpecStrategyCanary)
		(*in).DeepCopyInto(*out)
	}
	if in.ReconcileFrequency != nil {
		in, out := &in.ReconcileFrequency, &out.ReconcileFrequency
		*out = new(v1.Duration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetSpecStrategy.
func (in *ExtendedDaemonSetSpecStrategy) DeepCopy() *ExtendedDaemonSetSpecStrategy {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetSpecStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetSpecStrategyCanary) DeepCopyInto(out *ExtendedDaemonSetSpecStrategyCanary) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.Duration != nil {
		in, out := &in.Duration, &out.Duration
		*out = new(v1.Duration)
		**out = **in
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeAntiAffinityKeys != nil {
		in, out := &in.NodeAntiAffinityKeys, &out.NodeAntiAffinityKeys
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AutoPause != nil {
		in, out := &in.AutoPause, &out.AutoPause
		*out = new(ExtendedDaemonSetSpecStrategyCanaryAutoPause)
		(*in).DeepCopyInto(*out)
	}
	if in.AutoFail != nil {
		in, out := &in.AutoFail, &out.AutoFail
		*out = new(ExtendedDaemonSetSpecStrategyCanaryAutoFail)
		(*in).DeepCopyInto(*out)
	}
	if in.NoRestartsDuration != nil {
		in, out := &in.NoRestartsDuration, &out.NoRestartsDuration
		*out = new(v1.Duration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetSpecStrategyCanary.
func (in *ExtendedDaemonSetSpecStrategyCanary) DeepCopy() *ExtendedDaemonSetSpecStrategyCanary {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetSpecStrategyCanary)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetSpecStrategyCanaryAutoFail) DeepCopyInto(out *ExtendedDaemonSetSpecStrategyCanaryAutoFail) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.MaxRestarts != nil {
		in, out := &in.MaxRestarts, &out.MaxRestarts
		*out = new(int32)
		**out = **in
	}
	if in.MaxRestartsDuration != nil {
		in, out := &in.MaxRestartsDuration, &out.MaxRestartsDuration
		*out = new(v1.Duration)
		**out = **in
	}
	if in.CanaryTimeout != nil {
		in, out := &in.CanaryTimeout, &out.CanaryTimeout
		*out = new(v1.Duration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetSpecStrategyCanaryAutoFail.
func (in *ExtendedDaemonSetSpecStrategyCanaryAutoFail) DeepCopy() *ExtendedDaemonSetSpecStrategyCanaryAutoFail {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetSpecStrategyCanaryAutoFail)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetSpecStrategyCanaryAutoPause) DeepCopyInto(out *ExtendedDaemonSetSpecStrategyCanaryAutoPause) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.MaxRestarts != nil {
		in, out := &in.MaxRestarts, &out.MaxRestarts
		*out = new(int32)
		**out = **in
	}
	if in.MaxSlowStartDuration != nil {
		in, out := &in.MaxSlowStartDuration, &out.MaxSlowStartDuration
		*out = new(v1.Duration)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetSpecStrategyCanaryAutoPause.
func (in *ExtendedDaemonSetSpecStrategyCanaryAutoPause) DeepCopy() *ExtendedDaemonSetSpecStrategyCanaryAutoPause {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetSpecStrategyCanaryAutoPause)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetSpecStrategyRollingUpdate) DeepCopyInto(out *ExtendedDaemonSetSpecStrategyRollingUpdate) {
	*out = *in
	if in.MaxUnavailable != nil {
		in, out := &in.MaxUnavailable, &out.MaxUnavailable
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.MaxPodSchedulerFailure != nil {
		in, out := &in.MaxPodSchedulerFailure, &out.MaxPodSchedulerFailure
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.MaxParallelPodCreation != nil {
		in, out := &in.MaxParallelPodCreation, &out.MaxParallelPodCreation
		*out = new(int32)
		**out = **in
	}
	if in.SlowStartIntervalDuration != nil {
		in, out := &in.SlowStartIntervalDuration, &out.SlowStartIntervalDuration
		*out = new(v1.Duration)
		**out = **in
	}
	if in.SlowStartAdditiveIncrease != nil {
		in, out := &in.SlowStartAdditiveIncrease, &out.SlowStartAdditiveIncrease
		*out = new(intstr.IntOrString)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetSpecStrategyRollingUpdate.
func (in *ExtendedDaemonSetSpecStrategyRollingUpdate) DeepCopy() *ExtendedDaemonSetSpecStrategyRollingUpdate {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetSpecStrategyRollingUpdate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetStatus) DeepCopyInto(out *ExtendedDaemonSetStatus) {
	*out = *in
	if in.Canary != nil {
		in, out := &in.Canary, &out.Canary
		*out = new(ExtendedDaemonSetStatusCanary)
		(*in).DeepCopyInto(*out)
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ExtendedDaemonSetCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetStatus.
func (in *ExtendedDaemonSetStatus) DeepCopy() *ExtendedDaemonSetStatus {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonSetStatusCanary) DeepCopyInto(out *ExtendedDaemonSetStatusCanary) {
	*out = *in
	if in.Nodes != nil {
		in, out := &in.Nodes, &out.Nodes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonSetStatusCanary.
func (in *ExtendedDaemonSetStatusCanary) DeepCopy() *ExtendedDaemonSetStatusCanary {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonSetStatusCanary)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonsetSetting) DeepCopyInto(out *ExtendedDaemonsetSetting) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonsetSetting.
func (in *ExtendedDaemonsetSetting) DeepCopy() *ExtendedDaemonsetSetting {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonsetSetting)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExtendedDaemonsetSetting) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonsetSettingContainerSpec) DeepCopyInto(out *ExtendedDaemonsetSettingContainerSpec) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonsetSettingContainerSpec.
func (in *ExtendedDaemonsetSettingContainerSpec) DeepCopy() *ExtendedDaemonsetSettingContainerSpec {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonsetSettingContainerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonsetSettingList) DeepCopyInto(out *ExtendedDaemonsetSettingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ExtendedDaemonsetSetting, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonsetSettingList.
func (in *ExtendedDaemonsetSettingList) DeepCopy() *ExtendedDaemonsetSettingList {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonsetSettingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ExtendedDaemonsetSettingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonsetSettingSpec) DeepCopyInto(out *ExtendedDaemonsetSettingSpec) {
	*out = *in
	if in.Reference != nil {
		in, out := &in.Reference, &out.Reference
		*out = new(autoscalingv1.CrossVersionObjectReference)
		**out = **in
	}
	in.NodeSelector.DeepCopyInto(&out.NodeSelector)
	if in.Containers != nil {
		in, out := &in.Containers, &out.Containers
		*out = make([]ExtendedDaemonsetSettingContainerSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonsetSettingSpec.
func (in *ExtendedDaemonsetSettingSpec) DeepCopy() *ExtendedDaemonsetSettingSpec {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonsetSettingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtendedDaemonsetSettingStatus) DeepCopyInto(out *ExtendedDaemonsetSettingStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtendedDaemonsetSettingStatus.
func (in *ExtendedDaemonsetSettingStatus) DeepCopy() *ExtendedDaemonsetSettingStatus {
	if in == nil {
		return nil
	}
	out := new(ExtendedDaemonsetSettingStatus)
	in.DeepCopyInto(out)
	return out
}
