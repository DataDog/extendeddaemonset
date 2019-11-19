// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonsetreplicaset

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilserrors "k8s.io/apimachinery/pkg/util/errors"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/datadog/extendeddaemonset/pkg/config"
	"github.com/datadog/extendeddaemonset/pkg/controller/extendeddaemonsetreplicaset/conditions"
	"github.com/datadog/extendeddaemonset/pkg/controller/extendeddaemonsetreplicaset/strategy"
	"github.com/datadog/extendeddaemonset/pkg/controller/utils/enqueue"
)

var log = logf.Log.WithName("ExtendedDaemonSetReplicaSet")

// Add creates a new ExtendedDaemonSetReplicaSet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileExtendedDaemonSetReplicaSet{
		client:                  mgr.GetClient(),
		scheme:                  mgr.GetScheme(),
		recorder:                mgr.GetEventRecorderFor("ExtendedDaemonSetReplicaSet"),
		isNodeAffinitySupported: os.Getenv(config.NodeAffinityMatchSupportEnvVar) == "1",
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("extendeddaemonsetreplicaset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ExtendedDaemonSetReplicaSet
	err = c.Watch(&source.Kind{Type: &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ExtendedDaemonSet
	err = c.Watch(&source.Kind{Type: &datadoghqv1alpha1.ExtendedDaemonSet{}}, &enqueue.RequestForExtendedDaemonSetStatus{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ExtendedDaemonSetReplicaSet
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{},
	})
	if err != nil {
		return err
	}

	// watch also Nodes, to scale up or note the replicaset
	rsReconciler, ok := r.(*ReconcileExtendedDaemonSetReplicaSet)
	if !ok {
		return fmt.Errorf("unable to cast Reconciler to ReconcileExtendedDaemonSetReplicaSet")
	}
	nodeEventHandler := enqueue.NewRequestForAllReplicaSetFromNodeEvent(rsReconciler.client)
	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, nodeEventHandler)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileExtendedDaemonSetReplicaSet implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileExtendedDaemonSetReplicaSet{}

// ReconcileExtendedDaemonSetReplicaSet reconciles a ExtendedDaemonSetReplicaSet object
type ReconcileExtendedDaemonSetReplicaSet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder

	isNodeAffinitySupported bool
}

// Reconcile reads that state of the cluster for a ExtendedDaemonSetReplicaSet object and makes changes based on the state read
// and what is in the ExtendedDaemonSetReplicaSet.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileExtendedDaemonSetReplicaSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ExtendedDaemonSetReplicaSet")

	now := metav1.NewTime(time.Now())
	// Fetch the ExtendedDaemonSetReplicaSet replicaSetInstance
	replicaSetInstance, needReturn, err := r.retrievedReplicaSet(request)
	if needReturn {
		return reconcile.Result{}, err
	}

	lastResyncTimeStampCond := conditions.GetExtendedDaemonSetReplicaSetStatusCondition(&replicaSetInstance.Status, datadoghqv1alpha1.ConditionTypeLastFullSync)
	if lastResyncTimeStampCond != nil {
		nextSyncTS := lastResyncTimeStampCond.LastUpdateTime.Add(replicaSetInstance.Spec.Strategy.ReconcileFrequency.Duration)
		if nextSyncTS.After(now.Time) {
			requeueDuration := nextSyncTS.Sub(now.Time)
			reqLogger.V(1).Info("Reconcile, skip this resync", "requeueAfter", requeueDuration)
			return reconcile.Result{RequeueAfter: requeueDuration}, nil
		}
	}

	// First retrieve the Parent DDaemonset
	daemonsetInstance, err := r.getDaemonsetOwner(replicaSetInstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// retrieved and build information for the strategy
	strategyParams, err := r.buildStrategyParams(reqLogger, daemonsetInstance, replicaSetInstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// now apply the strategy depending on the ReplicaSet state
	strategyResult, err := r.applyStrategy(reqLogger, now, strategyParams)
	newStatus := strategyResult.NewStatus
	result := strategyResult.Result

	// for the reste of the actions we will try to execute as many actions as we can so we will store possible errors in a list
	// each action can be executed in parallel.
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	var desc string
	status := corev1.ConditionTrue
	if len(strategyResult.UnscheduledNodesDueToResourcesConstraints) > 0 {
		desc = fmt.Sprintf("nodes:%s", strings.Join(strategyResult.UnscheduledNodesDueToResourcesConstraints, ";"))
	} else {
		status = corev1.ConditionFalse
	}
	conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(newStatus, now, datadoghqv1alpha1.ConditionTypeUnschedule, status, desc, false, false)

	// start actions on pods
	lastPodDeletionCondition := conditions.GetExtendedDaemonSetReplicaSetStatusCondition(newStatus, datadoghqv1alpha1.ConditionTypePodDeletion)
	if lastPodDeletionCondition != nil && now.Sub(lastPodDeletionCondition.LastUpdateTime.Time) < 5*time.Second {
		result.RequeueAfter = 5 * time.Second
	} else {
		errs = append(errs, deletePods(reqLogger, r.client, strategyParams.PodByNodeName, strategyResult.PodsToDelete)...)
		if len(strategyResult.PodsToDelete) > 0 {
			conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(newStatus, now, datadoghqv1alpha1.ConditionTypePodDeletion, corev1.ConditionTrue, "pods deleted", false, true)
		}
	}

	lastPodCreationCondition := conditions.GetExtendedDaemonSetReplicaSetStatusCondition(newStatus, datadoghqv1alpha1.ConditionTypePodCreation)
	if lastPodCreationCondition != nil && now.Sub(lastPodCreationCondition.LastUpdateTime.Time) < replicaSetInstance.Spec.Strategy.ReconcileFrequency.Duration {
		result.RequeueAfter = replicaSetInstance.Spec.Strategy.ReconcileFrequency.Duration
	} else {
		errs = append(errs, createPods(reqLogger, r.client, r.scheme, r.isNodeAffinitySupported, replicaSetInstance, strategyResult.PodsToCreate)...)
		if len(strategyResult.PodsToCreate) > 0 {
			conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(newStatus, now, datadoghqv1alpha1.ConditionTypePodCreation, corev1.ConditionTrue, "pods created", false, true)
		}
	}

	err = utilserrors.NewAggregate(errs)
	conditions.UpdateErrorCondition(newStatus, now, err, "")
	conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(newStatus, now, datadoghqv1alpha1.ConditionTypeLastFullSync, corev1.ConditionTrue, "full sync", true, true)
	err = r.updateReplicaSet(replicaSetInstance, newStatus)
	reqLogger.V(1).Info("Reconcile end", "return", result, "error", err)
	return result, err
}

func (r *ReconcileExtendedDaemonSetReplicaSet) buildStrategyParams(logger logr.Logger, daemonset *datadoghqv1alpha1.ExtendedDaemonSet, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (*strategy.Parameters, error) {
	rsStatus := retrieveReplicaSetStatus(daemonset, replicaset.Name)

	// Retrieve the Node associated to the replicaset (with node selector)
	nodeList, podList, err := r.getPodAndNodeList(logger, daemonset, replicaset)
	if err != nil {
		return nil, err
	}

	strategyParams := &strategy.Parameters{
		Replicaset:       replicaset,
		ReplicaSetStatus: string(rsStatus),
		Logger:           logger.WithValues("strategy", rsStatus),
		NewStatus:        replicaset.Status.DeepCopy(),
	}
	var nodesFilter []string
	if daemonset.Status.Canary != nil {
		strategyParams.CanaryNodes = daemonset.Status.Canary.Nodes
		if daemonset.Status.ActiveReplicaSet == replicaset.Name {
			nodesFilter = strategyParams.CanaryNodes
		}
	}

	// Associate Pods to Nodes
	strategyParams.PodByNodeName, strategyParams.PodToCleanUp, strategyParams.UnscheduledPods = FilterAndMapPodsByNode(logger.WithValues("status", string(rsStatus)), replicaset, nodeList, podList, nodesFilter)

	return strategyParams, nil
}

func (r *ReconcileExtendedDaemonSetReplicaSet) applyStrategy(logger logr.Logger, now metav1.Time, strategyParams *strategy.Parameters) (*strategy.Result, error) {
	var strategyResult *strategy.Result
	var err error

	switch strategy.ReplicaSetStatus(strategyParams.ReplicaSetStatus) {
	case strategy.ReplicaSetStatusActive:
		logger.Info("manage deployment")
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(strategyParams.NewStatus, now, datadoghqv1alpha1.ConditionTypeActive, corev1.ConditionTrue, "", false, false)
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(strategyParams.NewStatus, now, datadoghqv1alpha1.ConditionTypeCanary, corev1.ConditionFalse, "", false, false)
		strategyResult, err = strategy.ManageDeployment(r.client, strategyParams)
	case strategy.ReplicaSetStatusCanary:
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(strategyParams.NewStatus, now, datadoghqv1alpha1.ConditionTypeCanary, corev1.ConditionTrue, "", false, false)
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(strategyParams.NewStatus, now, datadoghqv1alpha1.ConditionTypeActive, corev1.ConditionFalse, "", false, false)
		logger.Info("manage canary deployment")
		strategyResult, err = strategy.ManageCanaryDeployment(r.client, strategyParams)
	case strategy.ReplicaSetStatusUnknown:
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(strategyParams.NewStatus, now, datadoghqv1alpha1.ConditionTypeCanary, corev1.ConditionFalse, "", false, false)
		conditions.UpdateExtendedDaemonSetReplicaSetStatusCondition(strategyParams.NewStatus, now, datadoghqv1alpha1.ConditionTypeActive, corev1.ConditionFalse, "", false, false)
		logger.Info("ignore this replicaset, since it's not the replicas active or canary")
		strategyResult, err = strategy.ManageUnknow(r.client, strategyParams)
	}
	return strategyResult, err
}

func (r *ReconcileExtendedDaemonSetReplicaSet) getPodAndNodeList(logger logr.Logger, daemonset *datadoghqv1alpha1.ExtendedDaemonSet, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (*corev1.NodeList, *corev1.PodList, error) {
	var nodeList *corev1.NodeList
	var podList *corev1.PodList
	var err error
	nodeList, err = r.getNodeList(replicaset)
	if err != nil {
		logger.Error(err, "unable to list associated pods")
		return nodeList, podList, err
	}

	// Retrieve the Node associated to the replicaset
	podList, err = r.getPodList(daemonset)
	if err != nil {
		logger.Error(err, "unable to list associated pods")
		return nodeList, podList, err
	}

	var oldPodList *corev1.PodList
	oldPodList, err = r.getOldDaemonsetPodList(daemonset)
	if err != nil {
		logger.Error(err, "unable to list associated pods")
		return nodeList, podList, err
	}
	podList.Items = append(podList.Items, oldPodList.Items...)

	return nodeList, podList, nil
}

func (r *ReconcileExtendedDaemonSetReplicaSet) retrievedReplicaSet(request reconcile.Request) (*datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, bool, error) {
	replicaSetInstance := &datadoghqv1alpha1.ExtendedDaemonSetReplicaSet{}
	err := r.client.Get(context.TODO(), request.NamespacedName, replicaSetInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return nil, true, nil
		}
		// Error reading the object - requeue the request.
		return nil, true, err
	}
	return replicaSetInstance, false, nil
}

func (r *ReconcileExtendedDaemonSetReplicaSet) updateReplicaSet(replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, newStatus *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetStatus) error {
	// compare
	if !apiequality.Semantic.DeepEqual(&replicaset.Status, newStatus) {
		newRS := replicaset.DeepCopy()
		newRS.Status = *newStatus
		return r.client.Status().Update(context.TODO(), newRS)
	}
	return nil
}

func (r *ReconcileExtendedDaemonSetReplicaSet) getDaemonsetOwner(replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (*datadoghqv1alpha1.ExtendedDaemonSet, error) {
	ownerName, err := retrieveOwnerReference(replicaset)
	if err != nil {
		return nil, err
	}
	daemonsetInstance := &datadoghqv1alpha1.ExtendedDaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: ownerName, Namespace: replicaset.Namespace}, daemonsetInstance)
	if err != nil {
		return nil, err
	}
	return daemonsetInstance, nil
}

func (r *ReconcileExtendedDaemonSetReplicaSet) getPodList(ds *datadoghqv1alpha1.ExtendedDaemonSet) (*corev1.PodList, error) {
	podList := &corev1.PodList{}
	podSelector := labels.Set{datadoghqv1alpha1.ExtendedDaemonSetNameLabelKey: ds.Name}
	podListOptions := []client.ListOption{
		client.MatchingLabelsSelector{
			Selector: podSelector.AsSelectorPreValidated(),
		},
	}
	if err := r.client.List(context.TODO(), podList, podListOptions...); err != nil {
		return nil, err
	}
	return podList, nil
}

func (r *ReconcileExtendedDaemonSetReplicaSet) getNodeList(replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (*corev1.NodeList, error) {
	nodeList := &corev1.NodeList{}
	nodeSelector := labels.Set{}
	if replicaset.Spec.Selector != nil {
		nodeSelector = labels.Set(replicaset.Spec.Selector.MatchLabels)
	}
	listOptions := []client.ListOption{
		client.MatchingLabelsSelector{
			Selector: nodeSelector.AsSelectorPreValidated(),
		},
	}
	if err := r.client.List(context.TODO(), nodeList, listOptions...); err != nil {
		return nil, err
	}
	return nodeList, nil
}

func (r *ReconcileExtendedDaemonSetReplicaSet) getOldDaemonsetPodList(ds *datadoghqv1alpha1.ExtendedDaemonSet) (*corev1.PodList, error) {
	podList := &corev1.PodList{}

	oldDsName, ok := ds.GetAnnotations()[datadoghqv1alpha1.ExtendedDaemonSetOldDaemonsetAnnotationKey]
	if !ok {
		return podList, nil
	}

	oldDaemonset := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: ds.Namespace, Name: oldDsName}, oldDaemonset)
	if err != nil {
		if errors.IsNotFound(err) {
			return podList, nil
		}
		// Error reading the object - requeue the request.
		return nil, err
	}
	var podSelector labels.Set
	if oldDaemonset.Spec.Selector != nil {
		podSelector = labels.Set(oldDaemonset.Spec.Selector.MatchLabels)
	}
	podListOptions := []client.ListOption{
		client.MatchingLabelsSelector{
			Selector: podSelector.AsSelectorPreValidated(),
		},
	}
	if err = r.client.List(context.TODO(), podList, podListOptions...); err != nil {
		return nil, err
	}

	// filter by ownerreferences,
	// This is to prevent issue with label selector that match between DS and EDS
	var filterPods []corev1.Pod
	for id, pod := range podList.Items {
		selected := false
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == "DaemonSet" && ref.Name == oldDsName {
				selected = true
				break
			}
		}
		if selected {
			filterPods = append(filterPods, podList.Items[id])
		}
	}
	podList.Items = filterPods

	return podList, nil
}

func retrieveOwnerReference(obj *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (string, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "ExtendedDaemonSet" {
			return ref.Name, nil
		}
	}

	return "", fmt.Errorf("unable to retrieve the owner reference name")
}

func retrieveReplicaSetStatus(daemonset *datadoghqv1alpha1.ExtendedDaemonSet, replicassetName string) strategy.ReplicaSetStatus {
	switch daemonset.Status.ActiveReplicaSet {
	case "":
		return strategy.ReplicaSetStatusUnknown
	case replicassetName:
		return strategy.ReplicaSetStatusActive
	default:
		if daemonset.Status.Canary != nil && daemonset.Status.Canary.ReplicaSet == replicassetName {
			return strategy.ReplicaSetStatusCanary
		}
		return strategy.ReplicaSetStatusUnknown
	}
}
