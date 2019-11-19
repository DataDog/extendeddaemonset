// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	podutils "github.com/datadog/extendeddaemonset/pkg/controller/utils/pod"
	"github.com/go-logr/logr"
)

func createPods(logger logr.Logger, client client.Client, scheme *runtime.Scheme, podAffinitySupported bool, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, podsToCreate []string) []error {
	var errs []error
	var wg sync.WaitGroup
	errsChan := make(chan error, len(podsToCreate))
	for id := range podsToCreate {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			newPod := podutils.CreatePodFromDaemonSetReplicaSet(scheme, replicaset, podsToCreate[id], podAffinitySupported)
			logger.V(1).Info("Create pod", "name", newPod.GenerateName, "node", podsToCreate[id], "addAffinity", podAffinitySupported)
			err := client.Create(context.TODO(), newPod)
			if err != nil {
				logger.Error(err, "Create pod failed", "name", newPod.GenerateName)
				errsChan <- err
			}
		}(id)
	}
	go func() {
		wg.Wait()
		close(errsChan)
	}()

	for err := range errsChan {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func deletePods(logger logr.Logger, c client.Client, podByNodeName map[string]*corev1.Pod, nodeNames []string) []error {
	var errs []error
	var wg sync.WaitGroup
	errsChan := make(chan error, len(nodeNames))
	for _, nodeName := range nodeNames {
		wg.Add(1)
		go func(nodeName string) {
			defer wg.Done()
			logger.V(1).Info("Delete pod", "name", podByNodeName[nodeName].Name, "node", nodeName)
			err := c.Delete(context.TODO(), podByNodeName[nodeName])
			if err != nil {
				errsChan <- err
			}
		}(nodeName)
	}
	go func() {
		wg.Wait()
		close(errsChan)
	}()

	for err := range errsChan {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
