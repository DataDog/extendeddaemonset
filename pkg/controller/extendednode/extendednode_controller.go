// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendednode

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
)

var log = logf.Log.WithName("controller_extendednode")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ExtendedNode Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileExtendedNode{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("extendednode-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ExtendedNode
	err = c.Watch(&source.Kind{Type: &datadoghqv1alpha1.ExtendedNode{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileExtendedNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileExtendedNode{}

// ReconcileExtendedNode reconciles a ExtendedNode object
type ReconcileExtendedNode struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a ExtendedNode object and makes changes based on the state read
// and what is in the ExtendedNode.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileExtendedNode) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ExtendedNode")

	// Fetch the ExtendedNode instance
	instance := &datadoghqv1alpha1.ExtendedNode{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	newStatus := instance.Status.DeepCopy()
	newStatus.Error = ""
	if instance.Spec.Reference == nil || instance.Spec.Reference.Name == "" {
		newStatus.Error = "missing reference"

		newStatus.Error = fmt.Sprintf("missing reference in spec")
		newStatus.Status = datadoghqv1alpha1.ExtendedNodeStatusError
		return r.updateExtendedNode(instance, newStatus)
	}

	edsNodesList := &datadoghqv1alpha1.ExtendedNodeList{}
	if err = r.client.List(context.TODO(), edsNodesList, &client.ListOptions{Namespace: instance.Namespace}); err != nil {
		return r.updateExtendedNode(instance, newStatus)
	}

	nodesList := &corev1.NodeList{}
	if err = r.client.List(context.TODO(), nodesList); err != nil {
		newStatus.Status = datadoghqv1alpha1.ExtendedNodeStatusError
		newStatus.Error = fmt.Sprintf("unable to get nodes, err:%v", err)
	}

	var otherEdsNode string
	otherEdsNode, err = searchPossibleConflict(instance, nodesList, edsNodesList)
	if err != nil {
		newStatus.Status = datadoghqv1alpha1.ExtendedNodeStatusError
		newStatus.Error = fmt.Sprintf("conflict with another ExtendedNode: %s", otherEdsNode)
	}

	if newStatus.Error == "" {
		newStatus.Status = datadoghqv1alpha1.ExtendedNodeStatusValid
	}

	return r.updateExtendedNode(instance, newStatus)
}

func (r *ReconcileExtendedNode) updateExtendedNode(edsNode *datadoghqv1alpha1.ExtendedNode, newStatus *datadoghqv1alpha1.ExtendedNodeStatus) (reconcile.Result, error) {
	if apiequality.Semantic.DeepEqual(&edsNode.Status, newStatus) {
		return reconcile.Result{}, nil
	}
	newEdsNode := edsNode.DeepCopy()
	newEdsNode.Status = *newStatus
	err := r.client.Status().Update(context.TODO(), newEdsNode)

	return reconcile.Result{}, err
}

func searchPossibleConflict(instance *datadoghqv1alpha1.ExtendedNode, nodeList *corev1.NodeList, edsNodeList *datadoghqv1alpha1.ExtendedNodeList) (string, error) {
	var edsNodes edsNodeByCreationTimestampAndPhase
	for id := range edsNodeList.Items {
		edsNodes = append(edsNodes, &edsNodeList.Items[id])
	}
	sort.Sort(edsNodes)

	nodesAlreadySelected := map[string]string{}
	for _, node := range nodeList.Items {
		for _, edsNode := range edsNodes {
			selector, err2 := metav1.LabelSelectorAsSelector(&edsNode.Spec.NodeSelector)
			if err2 != nil {
				return "", err2
			}
			if selector.Matches(labels.Set(node.Labels)) {
				if edsNode.Name == instance.Name {
					if previousEdsNode, found := nodesAlreadySelected[node.Name]; found {
						return previousEdsNode, fmt.Errorf("extendedNode already assigned to the node %s", node.Name)
					}
				}
				nodesAlreadySelected[node.Name] = edsNode.Name
			}
		}
		nodesAlreadySelected[node.Name] = ""
	}
	return "", nil
}
