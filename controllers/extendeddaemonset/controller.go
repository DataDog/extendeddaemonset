// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package extendeddaemonset

import (
	"context"
	"fmt"
	"strings"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers/extendeddaemonsetreplicaset/scheduler"
	"github.com/DataDog/extendeddaemonset/pkg/controller/utils"
	"github.com/DataDog/extendeddaemonset/pkg/controller/utils/comparison"
	podutils "github.com/DataDog/extendeddaemonset/pkg/controller/utils/pod"
)

// Reconciler is the internal reconciler for ExtendedDaemonSet
type Reconciler struct {
	options  ReconcilerOptions
	client   client.Client
	scheme   *runtime.Scheme
	log      logr.Logger
	recorder record.EventRecorder
}

// ReconcilerOptions provides options read from command line
type ReconcilerOptions struct {
}

// NewReconciler returns a reconciler for DatadogAgent
func NewReconciler(options ReconcilerOptions, client client.Client, scheme *runtime.Scheme, log logr.Logger, recorder record.EventRecorder) (*Reconciler, error) {
	return &Reconciler{
		options:  options,
		client:   client,
		scheme:   scheme,
		log:      log,
		recorder: recorder,
	}, nil
}

// Reconcile reads that state of the cluster for a ExtendedDaemonSet object and makes changes based on the state read
// and what is in the ExtendedDaemonSet.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := r.log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
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

	if err = datadoghqv1alpha1.ValidateExtendedDaemonSetSpec(&instance.Spec); err != nil {
		return reconcile.Result{}, err
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

	_, result, err := r.updateInstanceWithCurrentRS(reqLogger, instance, currentRS, upToDateRS, podsCounter)
	result = utils.MergeResult(result, reconcile.Result{RequeueAfter: requeueAfter})
	return result, err
}

func (r *Reconciler) createNewReplicaSet(logger logr.Logger, daemonset *datadoghqv1alpha1.ExtendedDaemonSet) (reconcile.Result, error) {
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
	logger.Info("Creating a new ReplicaSet", "replicaSet.Namespace", newRS.Namespace, "replicaSet.Name", newRS.Name)

	err = r.client.Create(context.TODO(), newRS)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.recorder.Event(daemonset, corev1.EventTypeNormal, "Create ExtendedDaemonSetReplicaSet", fmt.Sprintf("%s/%s", newRS.Namespace, newRS.Name))
	return reconcile.Result{Requeue: true}, nil
}

// selectCurrentReplicaSet returns the replicaset that should be current
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
	dsAnnotations := daemonset.GetAnnotations()
	isEnded, requeueAfter = IsCanaryDeploymentEnded(daemonset.Spec.Strategy.Canary, upToDateRS, now)
	isPaused, _ := IsCanaryDeploymentPaused(dsAnnotations)
	isValid := IsCanaryDeploymentValid(dsAnnotations, upToDateRS.GetName())
	if isValid || (!isPaused && isEnded) {
		return upToDateRS, requeueAfter
	}

	return activeRS, requeueAfter
}

func (r *Reconciler) updateInstanceWithCurrentRS(logger logr.Logger, daemonset *datadoghqv1alpha1.ExtendedDaemonSet, current, upToDate *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, podsCounter podsCounterType) (*datadoghqv1alpha1.ExtendedDaemonSet, reconcile.Result, error) {
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

	var updateDaemonsetSpec bool
	// If the deployment is in Canary phase, then update status (and spec as needed)
	if daemonset.Spec.Strategy.Canary != nil {
		switch {
		case IsCanaryDeploymentFailed(daemonset.GetAnnotations()):
			// Canary deployment is no longer needed because it was marked as failed
			// This `if` block needs to be first to respect Failed Canary state until annotation is removed
			newDaemonset.Status.Canary = nil
			newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryFailed
			// Restore active replicaset template. Note: this requires a full daemonset update
			newDaemonset.Spec.Template = current.Spec.Template
			updateDaemonsetSpec = true
		case current.Name == upToDate.Name:
			// Canary deployment is no longer needed because it completed without issue
			newDaemonset.Status.Canary = nil
			newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateRunning
		default:
			// Else compute the Canary status
			if newDaemonset.Status.Canary == nil {
				newDaemonset.Status.Canary = &datadoghqv1alpha1.ExtendedDaemonSetStatusCanary{}
			}
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

			isPaused, reason := IsCanaryDeploymentPaused(daemonset.GetAnnotations())
			if isPaused {
				newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanaryPaused
				newDaemonset.Status.Reason = reason
			} else {
				newDaemonset.Status.State = datadoghqv1alpha1.ExtendedDaemonSetStatusStateCanary
			}
		}
	}

	// Check if newDaemonset differs from existing daemonset, and update if so
	if !apiequality.Semantic.DeepEqual(daemonset, newDaemonset) {
		if updateDaemonsetSpec {
			// Make and use a copy because undesired behaviors occur when making two update calls
			newDaemonsetCopy := newDaemonset.DeepCopy()
			if err := r.client.Update(context.TODO(), newDaemonsetCopy); err != nil {
				return newDaemonsetCopy, reconcile.Result{}, err
			}

			// This ensures that the first client update respects the desired new status
			newDaemonsetCopy.Status = newDaemonset.Status
			if err := r.client.Status().Update(context.TODO(), newDaemonsetCopy); err != nil {
				return newDaemonsetCopy, reconcile.Result{}, err
			}
			return newDaemonsetCopy, reconcile.Result{}, nil
		}

		if err := r.client.Status().Update(context.TODO(), newDaemonset); err != nil {
			return newDaemonset, reconcile.Result{}, err
		}
		return newDaemonset, reconcile.Result{}, nil
	}

	return newDaemonset, reconcile.Result{}, nil
}

func (r *Reconciler) selectNodes(logger logr.Logger, daemonsetSpec *datadoghqv1alpha1.ExtendedDaemonSetSpec, replicaset *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet, canaryStatus *datadoghqv1alpha1.ExtendedDaemonSetStatusCanary) error {
	// create a Fake pod from the current replicaset.spec.template
	newPod, _ := podutils.CreatePodFromDaemonSetReplicaSet(r.scheme, replicaset, nil, nil, false)

	nodeList := &corev1.NodeList{}

	listOptions := []client.ListOption{}
	if daemonsetSpec.Strategy.Canary.NodeSelector != nil {
		selector, err := utils.ConvertLabelSelector(logger, daemonsetSpec.Strategy.Canary.NodeSelector)
		if err != nil {
			logger.Error(err, "Failed to parse label selector")
		} else {
			listOptions = append(listOptions, &client.MatchingLabelsSelector{
				Selector: selector,
			})
		}
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

	// Filter Nodes Unschedulable
	for _, node := range nodeList.Items {
		found := false
		var id int
		for id = range currentNodes {
			if node.Name == currentNodes[id] {
				found = true
				break
			}
		}

		if !found {
			continue
		}

		if !scheduler.CheckNodeFitness(logger.WithValues("filter", "Nodes Unschedulabled"), newPod, &node) {
			currentNodes = append(currentNodes[:id], currentNodes[id+1:]...)
		}
	}

	// Look for other nodes to use as canary
	if len(currentNodes) < nbCanaryPod {
		antiAffinityKeysValues := make(map[string]int)

		// Look for the values of the labels set as NodeAntiAffinityKeys of the nodes already selected as canary
		if len(daemonsetSpec.Strategy.Canary.NodeAntiAffinityKeys) != 0 {
			for _, node := range nodeList.Items {

				antiAffinityKeysValue := getAntiAffinityKeysValue(&node, daemonsetSpec)
				if _, found := antiAffinityKeysValues[antiAffinityKeysValue]; !found {
					antiAffinityKeysValues[antiAffinityKeysValue] = 0
				}

				for _, currentNode := range currentNodes {
					if node.Name == currentNode {
						antiAffinityKeysValues[antiAffinityKeysValue]++
						break
					}
				}
			}
		}

		// Look for new canary nodes
		for _, node := range nodeList.Items {
			// Check if that node is already selected
			alreadySelected := false
			for _, currentNode := range currentNodes {
				if node.Name == currentNode {
					alreadySelected = true
					break
				}
			}
			if alreadySelected {
				continue
			}

			// Ensure that the selected canary nodes are evenly chosen regarding the value of their labels selected by the `NodeAntiAffinityKeys` canary property
			//
			// For example, if a cluster has 100 nodes labeled `service=A` and 10 nodes labeled `service=B` and we have to choose 4 canary nodes, we want to choose 2 canary nodes labeled `service=B` and 2 canary nodes labeled `service=B`.
			// For that purpose, we use the `antiAffinityKeysValues` map that count the number of selected nodes per label value.
			// In our example, that map would have two entries: `A` and `B`.
			// We want a maximum of `nb_canaries / nb_different_values` (4/2=2) for each value.
			// So, we want to reject a node if it would make the selection unbalanced, i.e. if
			// antiAffinityKeysValues[getAntiAffinityKeysValue(&node, daemonsetSpec)] >= nbCanaryPod / len(antiAffinityKeysValues)
			//
			// In the above mathematical expression, we want the division result to be rounded up.
			// In our example, if we want 5 canary nodes, we will want 3 nodes labeled `service=A` and 2 nodes labeled `service=B` or the other way round
			// so, we want to reject nodes as soon as the number of selected nodes with that label value exceeds 3 = ceil(5/2)
			//
			// An efficient way to compute `ceil(a/b)` with only integer computing is to compute `(a+b-1)/b`.
			if len(daemonsetSpec.Strategy.Canary.NodeAntiAffinityKeys) != 0 {
				antiAffinityKeysValue := getAntiAffinityKeysValue(&node, daemonsetSpec)
				if nb := antiAffinityKeysValues[antiAffinityKeysValue]; nb >= (nbCanaryPod+len(antiAffinityKeysValues)-1)/len(antiAffinityKeysValues) {
					continue
				}
				antiAffinityKeysValues[antiAffinityKeysValue]++
			}

			currentNodes = append(currentNodes, node.Name)
			// All nodes are found. We can exit now!
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

func getAntiAffinityKeysValue(node *corev1.Node, daemonsetSpec *datadoghqv1alpha1.ExtendedDaemonSetSpec) string {
	values := make([]string, 0, len(daemonsetSpec.Strategy.Canary.NodeAntiAffinityKeys))
	for _, antiAffinityKey := range daemonsetSpec.Strategy.Canary.NodeAntiAffinityKeys {
		values = append(values, node.Labels[antiAffinityKey])
	}
	return strings.Join(values, "$")
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

func (r *Reconciler) cleanupReplicaSet(logger logr.Logger, rsList *datadoghqv1alpha1.ExtendedDaemonSetReplicaSetList, current, updatetodate *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) error {
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
