// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/datadog/extendeddaemonset/pkg/generated/clientset/versioned/typed/datadoghq/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeDatadoghqV1alpha1 struct {
	*testing.Fake
}

func (c *FakeDatadoghqV1alpha1) ExtendedDaemonSets(namespace string) v1alpha1.ExtendedDaemonSetInterface {
	return &FakeExtendedDaemonSets{c, namespace}
}

func (c *FakeDatadoghqV1alpha1) ExtendedDaemonSetReplicaSets(namespace string) v1alpha1.ExtendedDaemonSetReplicaSetInterface {
	return &FakeExtendedDaemonSetReplicaSets{c, namespace}
}

func (c *FakeDatadoghqV1alpha1) ExtendedNodes(namespace string) v1alpha1.ExtendedNodeInterface {
	return &FakeExtendedNodes{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeDatadoghqV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
