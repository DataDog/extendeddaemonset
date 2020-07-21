// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonset

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilserrors "k8s.io/apimachinery/pkg/util/errors"
	intstrutil "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/controller/extendeddaemonsetreplicaset/scheduler"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils/comparison"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils/enqueue"
	podutils "github.com/datadog/extendeddaemonset/pkg/controller/utils/pod"
)

var log = logf.Log.WithName("ExtendedDaemonSet")

// Add creates a new ExtendedDaemonSet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcileExtendedDaemonSet {
	return &ReconcileExtendedDaemonSet{client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetEventRecorderFor("ExtendedDaemonSet")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("extendeddaemonset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ExtendedDaemonSet
	err = c.Watch(&source.Kind{Type: &datadoghqv1alpha1.ExtendedDaemonSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource ExtendedDaemonSetReplicaSet and requeue the owner ExtendedDaemonSet
	err = c.Watch(&source.Kind{Type: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &datadoghqv1alpha1.ExtendedDaemonSet{},
	})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ExtendedDaemonSet
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &enqueue.RequestForExtendedDaemonSetLabel{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileExtendedDaemonSet implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileExtendedDaemonSet{}

// ReconcileExtendedDaemonSet reconciles a ExtendedDaemonSet object
type ReconcileExtendedDaemonSet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a ExtendedDaemonSet object and makes changes based on the state read
// and what is in the ExtendedDaemonSet.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileExtendedDaemonSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ExtendedDaemonSet")
	now := time.Now()
	// Fetch the ExtendedDaemonSet instance
	instance := &datadoghqv1alpha1.ExtendedDaemonSet{}
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

	if !datadoghqv1alpha1.IsDefaultedExtendedDaemonSet(instance) {
		reqLogger.Info("Defaulting values")
		defaultedInstance := datadoghqv1alpha1.DefaultExtendedDaemonSet(instance)
		err = r.client.Update(context.TODO(), defaultedInstance)
		if err != nil {
			reqLogger.Error(err, "failed to update ExtendedDaemonSet")
			return reconcile.Result{}, err
		}
		// ExtendedDaemonSet is now defaulted return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// counter for status
	var podsCounter podsCounterType

	// ExtendedDaemonSetReplicaSet attached to this instance
	replicaSetList := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSetList{}
	selector := labels.Set{
		datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey: request.Name,
	}
	listOpts := []client.ListOption{
		&client.MatchingLabelsSelector{Selector: selector.AsSelectorPreValidated()},
	}
	err = r.client.List(context.TODO(), replicaSetList, listOpts...)
	if err != nil {
		return reconcile.Result{}, err
	}
	var upToDateRS *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
	var activeRS *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
	for id, rs := range replicaSetList.Items {
		podsCounter.Ready += rs.Status.Ready
		podsCounter.Current += rs.Status.Available

		// Check if ReplicaSet is currently active
		if rs.Name == instance.Status.ActiveReplicaSet {
			activeRS = &replicaSetList.Items[id]
		}

		// Check if ReplicaSet matches the ExtendedDaemonset Spec
		if comparison.IsReplicaSetUpToDate(&rs, instance) {
			upToDateRS = rs.DeepCopy()
		}
	}

	if upToDateRS == nil {
		// If there is no ReplicaSet that matches the EDS Spec, create a new one and return to apply the reconcile loop again
		return r.createNewReplicaSet(reqLogger, instance)
	}

	// Select the ReplicaSet that should be current
	currentRS, requeueAfter := selectCurrentReplicaSet(instance, activeRS, upToDateRS, now)

	// Remove all ReplicaSets if not used anymore
	if err = r.cleanupReplicaSet(reqLogger, replicaSetList, currentRS, upToDateRS); err != nil {
		return reconcile.Result{RequeueAfter: requeueAfter}, nil
	}

	_, result, err := r.updateStatusWithNewRS(reqLogger, instance, currentRS, upToDateRS, podsCounter)
	result = utils.MergeResult(result, reconcile.Result{RequeueAfter: requeueAfter})
	return result, err
}

func (r *ReconcileExtendedDaemonSet) createNewReplicaSet(logger logr.Logger, daemonset *datadoghqv1alpha1.ExtendedDaemonSet) (reconcile.Result, error) {
	var err error
	// replicaSet up to date didn't exist yet, new to create one
	var newRS *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet
	if newRS, err = newReplicaSetFromInstance(daemonset); err != nil {
		return reconcile.Result{}, err
	}
	// Set ExtendedDaemonSet instance as the owner and controller
	if err = controllerutil.SetControllerReference(daemonset, newRS, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	err = r.client.Create(context.TODO(), newRS)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.recorder.Event(daemonset, corev1.EventTypeNormal, "Create ExtendedDaemonSetReplicaSet", fmt.Sprintf("%s/%s", newRS.Namespace, newRS.Name))

	return reconcile.Result{Requeue: true}, nil
}

// selectCurrentReplicaSet selects the replicaset that should be active
func selectCurrentReplicaSet(daemonset *datadoghqv1alpha1.ExtendedDaemonSet, activeRS, upToDateRS *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, now time.Time) (*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, time.Duration) {
	var requeueAfter time.Duration

	// If active and latest ReplicaSets are the same, nothing to do
	if activeRS == upToDateRS {
		return activeRS, requeueAfter
	}

	// If activeRS is nil (this can occur when an ERS exists while the operator is re-deployed), then use the latest ReplicaSet
	if activeRS == nil {
		return upToDateRS, requeueAfter
	}

	// If there is no Canary phase, then use the latest ReplicaSet
	if daemonset.Spec.Strategy.Canary == nil {
		return upToDateRS, requeueAfter
	}

	// If in Canary phase, then only update ReplicaSet if it has ended or been declared valid
	var isEnded bool
	isEnded, requeueAfter = IsCanaryPhaseEnded(daemonset.Spec.Strategy.Canary, upToDateRS, now)
	isPaused := IsCanaryPhasePaused(daemonset.Spec.Strategy.Canary)
	isValid := IsCanaryDeploymentValid(daemonset.GetAnnotations(), upToDateRS.GetName())
	if isValid || (!isPaused && isEnded) {
		return upToDateRS, requeueAfter
	}

	return activeRS, requeueAfter
}

func (r *ReconcileExtendedDaemonSet) updateStatusWithNewRS(logger logr.Logger, daemonset *datadoghqv1alpha1.ExtendedDaemonSet, current, upToDate *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, podsCounter podsCounterType) (*datadoghqv1alpha1.ExtendedDaemonSet, reconcile.Result, error) {
	newDaemonset := daemonset.DeepCopy()
	newDaemonset.Status.Current = podsCounter.Current
	newDaemonset.Status.Ready = podsCounter.Ready
	if current != nil {
		newDaemonset.Status.ActiveReplicaSet = current.Name
		newDaemonset.Status.Desired = current.Status.Desired
		newDaemonset.Status.UpToDate = current.Status.Available
		newDaemonset.Status.Available = current.Status.Available
		newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateRunning
		newDaemonset.Status.IgnoredUnresponsiveNodes = current.Status.IgnoredUnresponsiveNodes
	}

	// If the deployment is in Canary phase, then update status, strategy and state as needed
	if daemonset.Spec.Strategy.Canary != nil {
		if newDaemonset.Status.Canary == nil {
			newDaemonset.Status.Canary = &datadoghqv1alpha1.ExtendedDaemonSetStatusCanary{}
		}
		if current.Name == upToDate.Name {
			// Canary deployment is no longer needed because it completed without issue
			newDaemonset.Status.Canary = nil
			newDaemonset.Spec.Strategy.Canary = nil
			newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateRunning
			// Make sure Reason is empty in case it was populated by Canary failure
			newDaemonset.Status.Reason = ""
		} else if isFailed, reason := IsCanaryDeploymentFailed(daemonset.Spec.Strategy.Canary); isFailed {
			// Canary deployment is no longer needed because it was marked as failed
			newDaemonset.Status.Canary = nil
			newDaemonset.Spec.Strategy.Canary = nil
			newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed
			newDaemonset.Status.Reason = reason
		} else {
			// Else compute the Canary status
			newDaemonset.Status.Desired += upToDate.Status.Desired
			newDaemonset.Status.UpToDate += upToDate.Status.Available
			newDaemonset.Status.Available += upToDate.Status.Available
			newDaemonset.Status.IgnoredUnresponsiveNodes += upToDate.Status.IgnoredUnresponsiveNodes

			newDaemonset.Status.Canary.ReplicaSet = upToDate.Name

			nbCanaryPod, err := intstrutil.GetValueFromIntOrPercent(daemonset.Spec.Strategy.Canary.Replicas, int(daemonset.Status.Desired), true)
			if err != nil {
				logger.Error(err, "unable to select Nodes for canary")
				return newDaemonset, reconcile.Result{}, err
			}

			if nbCanaryPod != len(newDaemonset.Status.Canary.Nodes) {
				if err = r.selectNodes(logger, &newDaemonset.Spec, upToDate, newDaemonset.Status.Canary); err != nil {
					logger.Error(err, "unable to select Nodes for canary")
					return newDaemonset, reconcile.Result{}, err
				}
			}
			newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanary
		}
	}

	// compare
	if !apiequality.Semantic.DeepEqual(daemonset, newDaemonset) {
		if err := r.client.Status().Update(context.TODO(), newDaemonset); err != nil {
			return newDaemonset, reconcile.Result{}, err
		}
		return newDaemonset, reconcile.Result{}, nil
	}

	return newDaemonset, reconcile.Result{}, nil
}

func (r *ReconcileExtendedDaemonSet) selectNodes(logger logr.Logger, daemonsetSpec *datadoghqv1alpha1.ExtendedDaemonSetSpec, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, canaryStatus *datadoghqv1alpha1.ExtendedDaemonSetStatusCanary) error {
	// create a Fake pod from the current replicaset.spec.template
	newPod, _ := podutils.CreatePodFromDaemonSetReplicaSet(r.scheme, replicaset, nil, nil, false)

	nodeList := &corev1.NodeList{}
	nodeSelector := labels.Set{}
	if replicaset.Spec.Selector != nil {
		nodeSelector = labels.Set(replicaset.Spec.Selector.MatchLabels)
	}
	listOptions := []client.ListOption{
		&client.MatchingLabelsSelector{Selector: nodeSelector.AsSelectorPreValidated()},
	}
	err := r.client.List(context.TODO(), nodeList, listOptions...)
	if err != nil {
		return err
	}
	var currentNodes []string
	if canaryStatus != nil {
		currentNodes = canaryStatus.Nodes
	}

	nbCanaryPod, err := intstrutil.GetValueFromIntOrPercent(daemonsetSpec.Strategy.Canary.Replicas, int(replicaset.Status.Desired), true)
	if err != nil {
		return err
	}
	if len(currentNodes) < nbCanaryPod {
		for _, node := range nodeList.Items {
			found := false
			var id int
			for id = range currentNodes {
				if node.Name == currentNodes[id] {
					found = true
					break
				}
			}
			// Filter Nodes Unschedulabled
			if !scheduler.CheckNodeFitness(logger.WithValues("filter", "Nodes Unschedulabled"), newPod, &node, false) {
				if found {
					currentNodes = append(currentNodes[:id], currentNodes[id+1:]...)
				}
				continue
			}
			if !found {
				currentNodes = append(currentNodes, node.Name)
			}
			if len(currentNodes) == nbCanaryPod {
				logger.V(1).Info("All nodes were found")
				break
			}
		}
	}

	canaryStatus.Nodes = currentNodes
	if len(canaryStatus.Nodes) < nbCanaryPod {
		return fmt.Errorf("unable to select enough node for canary, current: %d, wanted: %d", len(canaryStatus.Nodes), nbCanaryPod)
	}
	return nil
}

func newReplicaSetFromInstance(daemonset *datadoghqv1alpha1.ExtendedDaemonSet) (*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, error) {
	labels := map[string]string{
		datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey: daemonset.Name,
	}
	for key, val := range daemonset.Labels {
		labels[key] = val
	}
	rs := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", daemonset.Name),
			Namespace:    daemonset.Namespace,
			Labels:       labels,
			Annotations:  daemonset.Annotations,
		},
		Spec: datadoghqv1alpha1.ExtendedDaemonSetReplicaSetSpec{
			Selector: daemonset.Spec.Selector.DeepCopy(),
			Template: *daemonset.Spec.Template.DeepCopy(),
		},
	}

	hash, err := comparison.SetMD5PodTemplateSpecAnnotation(rs, daemonset)
	rs.Spec.TemplateGeneration = hash
	return rs, err
}

func (r *ReconcileExtendedDaemonSet) cleanupReplicaSet(logger logr.Logger, rsList *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetList, current, updatetodate *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) error {
	var wg sync.WaitGroup
	errsChan := make(chan error, len(rsList.Items))
	for id, rs := range rsList.Items {
		if current == nil {
			continue
		}
		if rs.Name == current.Name {
			continue
		}
		if updatetodate != nil && rs.Name == updatetodate.Name {
			continue
		}
		if rs.DeletionTimestamp != nil {
			// already deleted
			continue
		}
		wg.Add(1)
		func(obj *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) {
			defer wg.Done()

			// TODO check if pods is still not attached to this eds.
			podList, err := getPodListFromReplicaSet(r.client, obj)
			if err != nil {
				errsChan <- err
				return
			}
			if podList == nil {
				errsChan <- fmt.Errorf("unable to get podList from: %s", obj.Name)
			}
			if len(podList.Items) == 0 {
				logger.Info("Delete replicaset", "replicaset_name", obj.Name)
				err := r.client.Delete(context.TODO(), obj)
				if err != nil {
					errsChan <- err
				}
			}
		}(&rsList.Items[id])
	}
	go func() {
		wg.Wait()
		close(errsChan)
	}()
	var errs []error
	for err := range errsChan {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return utilserrors.NewAggregate(errs)
}

type podsCounterType struct {
	Current int32
	Ready   int32
}
