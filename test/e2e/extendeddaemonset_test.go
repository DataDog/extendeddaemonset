// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package e2e

import (
	goctx "context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	apis "github.com/datadog/extendeddaemonset/pkg/apis"
	datadoghqv1alpha1 "github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
	"github.com/google/go-cmp/cmp"

	"github.com/prometheus/common/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	// "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"

	utils "github.com/datadog/extendeddaemonset/test/e2e/utils"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 360
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestEDS(t *testing.T) {
	extendeddaemonsetList := &datadoghqv1alpha1.ExtendedDaemonSetList{}

	if err := framework.AddToFrameworkScheme(apis.AddToScheme, extendeddaemonsetList); err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	nodes, err := framework.Global.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes.Items {
		found := false
		for _, taint := range node.Spec.Taints {
			if taint.Value == "app" && taint.Key == "node" {
				found = true
				break
			}
		}
		if found {
			continue
		}
		node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
			Effect: corev1.TaintEffectNoSchedule,
			Key:    "node",
			Value:  "app",
		})
		_, err = framework.Global.KubeClient.CoreV1().Nodes().Update(&node)
		if err != nil {
			t.Fatal(err)
		}
	}
	// run subtests
	t.Run("extendeddaemonset-group", func(t *testing.T) {
		t.Run("Initial-Deployment", InitialDeployment)
		//t.Run("MigrationFromDaemonSet", MigrationFromDaemonSet)
		t.Run("Use ExtendedDaemonsetSetting", UseExtendedDaemonsetSetting)
	})
}

func InitialDeployment(t *testing.T) {
	namespace, ctx, f := initTestFwkResources(t, "extendeddaemonset")
	defer ctx.Cleanup()

	name := "foo"
	maxUnavailable := intstr.FromInt(2)
	intString1 := intstr.FromInt(1)
	newOptions := &utils.NewExtendedDaemonsetOptions{
		CanaryStrategy: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
			Duration: &metav1.Duration{Duration: 5 * time.Minute},
			Replicas: &intString1,
		},
		RollingUpdate: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyRollingUpdate{
			MaxUnavailable:         &maxUnavailable,
			MaxParallelPodCreation: datadoghqv1alpha1.NewInt32(20),
		},
	}
	daemonset := utils.NewExtendedDaemonset(namespace, name, fmt.Sprintf("k8s.gcr.io/pause:%s", "latest"), newOptions)
	err := f.Client.Create(goctx.TODO(), daemonset, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	isOK := func(dd *datadoghqv1alpha1.ExtendedDaemonSet) (bool, error) {
		if dd.Status.ActiveReplicaSet != "" {
			return true, nil
		}
		return false, nil
	}
	err = utils.WaitForFuncOnExtendedDaemonset(t, f.Client, namespace, name, isOK, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// get ExtendedDaemonset
	daemonsetKey := dynclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err = f.Client.Get(goctx.TODO(), daemonsetKey, daemonset)
	if err != nil {
		t.Fatal(err)
	}

	// check if 2 the replicaset was created properly and the status is ok
	t.Logf("CELENE beginning replicaset check")
	isReplicaSetOK := func(rs *datadoghqv1alpha1.ExtendedDaemonSetReplicaSet) (bool, error) {
		t.Logf("%s %d %d %d", rs.Status.Status, rs.Status.Desired, rs.Status.Ready, rs.Status.Available)
		if rs.Status.Status != "active" {
			return false, nil
		}

		if rs.Status.Desired != 3 && rs.Status.Desired != rs.Status.Ready && rs.Status.Ready != rs.Status.Available {
			return false, nil
		}
		return true, nil
	}
	err = utils.WaitForFuncOnExtendedDaemonsetReplicaSet(t, f.Client, namespace, daemonset.Status.ActiveReplicaSet, isReplicaSetOK, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// // update the Extendeddaemonset and check that the status is updated to reflect Canary deployment
	// t.Logf("CELENE beginning Canary status check")
	// updateImage := func(eds *datadoghqv1alpha1.ExtendedDaemonSet) {
	// 	updatedImageTag := "3.1"
	// 	eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("k8s.gcr.io/pause:%s", updatedImageTag)
	// }
	// err = utils.UpdateExtendedDaemonSetFunc(f, namespace, daemonset.Name, updateImage, retryInterval, timeout)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// isUpdated := func(dd *datadoghqv1alpha1.ExtendedDaemonSet) (bool, error) {
	// 	if dd.Status.Canary != nil && dd.Status.Canary.ReplicaSet != "" {
	// 		return true, nil
	// 	}
	// 	return false, nil
	// }
	// err = utils.WaitForFuncOnExtendedDaemonset(t, f.Client, namespace, name, isUpdated, retryInterval, timeout)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// update the Extendeddaemonset and check that Canary autopauses when a canary pod restarts
	t.Logf("CELENE beginning Canary autopause check")
	updateImage := func(eds *datadoghqv1alpha1.ExtendedDaemonSet) {
		updatedImageTag := "3.1" // CELENE
		eds.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("k8s.gcr.io/pause:%s", updatedImageTag)

		// set low resource limits so pod will restart
		// resourceLimits := corev1.ResourceRequirements{
		// 	Limits: corev1.ResourceList{
		// 		// corev1.ResourceCPU:    resource.MustParse("0.001"),
		// 		corev1.ResourceMemory: resource.MustParse("1M"),
		// 	},
		// }
		// eds.Spec.Template.Spec.Containers[0].Resources = resourceLimits
	}
	t.Logf("CELENE updating EDS")
	err = utils.UpdateExtendedDaemonSetFunc(f, namespace, daemonset.Name, updateImage, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// t.Logf("CELENE checking EDS update")
	// edsTest := &datadoghqv1alpha1.ExtendedDaemonSet{}
	// err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: daemonset.Name, Namespace: namespace}, edsTest)
	// if err != nil {
	// 	t.Logf("CELENE failed to get updated EDS")
	// }
	// resourceLimits := corev1.ResourceRequirements{
	// 	Limits: corev1.ResourceList{
	// 		// corev1.ResourceCPU:    resource.MustParse("0.001"),
	// 		corev1.ResourceMemory: resource.MustParse("1M"),
	// 	},
	// }
	// if diff := cmp.Diff(resourceLimits, edsTest.Spec.Template.Spec.Containers[0].Resources); diff != "" {
	// 	t.Logf("CELENE output of EDS diff:")
	// 	t.Logf(diff)
	// } else {
	// 	t.Logf("CELENE no diff between EDSs")
	// }

	// isPaused := func(dd *datadoghqv1alpha1.ExtendedDaemonSet) (bool, error) {
	// 	if val, ok := dd.Annotations[datadoghqv1alpha1.ExtendedDaemonSetCanaryPausedAnnotationKey]; ok {
	// 		return val == "true", nil
	// 	}
	// 	return false, nil
	// }

	// t.Logf("CELENE checking isPaused")
	// err = utils.WaitForFuncOnExtendedDaemonset(t, f.Client, namespace, name, isPaused, retryInterval, timeout)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	isUpdated := func(dd *datadoghqv1alpha1.ExtendedDaemonSet) (bool, error) {
		if dd.Status.Canary != nil && dd.Status.Canary.ReplicaSet != "" {
			return true, nil
		}
		return false, nil
	}
	err = utils.WaitForFuncOnExtendedDaemonset(t, f.Client, namespace, name, isUpdated, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)

}

func MigrationFromDaemonSet(t *testing.T) {
	namespace, ctx, f := initTestFwkResources(t, "extendeddaemonset")
	defer ctx.Cleanup()

	oldDaemonsetName := "old-ds"
	oldDaemonset := utils.NewDaemonset(namespace, oldDaemonsetName, fmt.Sprintf("k8s.gcr.io/pause:%s", "latest"), nil)
	err := f.Client.Create(goctx.TODO(), oldDaemonset, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	isDSOK := func(ds *appsv1.DaemonSet) (bool, error) {
		if ds.Status.NumberReady == 3 {
			return true, nil
		}
		return false, nil
	}
	err = utils.WaitForFuncOnDaemonset(t, f.Client, namespace, oldDaemonsetName, isDSOK, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// test the upgrade procedure

	// update the daemonset spec in order to avoir that the Daemonset controller try to recreate pods
	updateDaemonset := func(ds *appsv1.DaemonSet) {
		ds.Spec.UpdateStrategy.Type = "OnDelete"
		if ds.Spec.Template.Spec.Affinity == nil {
			ds.Spec.Template.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{},
			}
		}
		ds.Spec.Template.Spec.Tolerations = nil
	}
	err = utils.UpdateDaemonSetFunc(f, namespace, oldDaemonsetName, updateDaemonset, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	isUpdated := func(ds *appsv1.DaemonSet) (bool, error) {
		if ds.Spec.UpdateStrategy.Type != "OnDelete" {
			return false, nil
		}

		if ds.Status.NumberMisscheduled == 0 {
			return false, nil
		}
		return true, nil
	}
	err = utils.WaitForFuncOnDaemonset(t, f.Client, namespace, oldDaemonsetName, isUpdated, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Now we can create the ExtendedDaemonset and provide the Daemonset old name as annotation
	name := "foo"
	intString1 := intstr.FromInt(1)
	newOptions := &utils.NewExtendedDaemonsetOptions{
		CanaryStrategy: &datadoghqv1alpha1.ExtendedDaemonSetSpecStrategyCanary{
			Duration: &metav1.Duration{Duration: 1 * time.Minute},
			Replicas: &intString1,
		},
		ExtraAnnotations: map[string]string{
			datadoghqv1alpha1.ExtendedDaemonSetOldDaemonsetAnnotationKey: oldDaemonsetName,
		},
	}
	daemonset := utils.NewExtendedDaemonset(namespace, name, fmt.Sprintf("k8s.gcr.io/pause:%s", "3.1"), newOptions)
	err = f.Client.Create(goctx.TODO(), daemonset, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	isOK := func(dd *datadoghqv1alpha1.ExtendedDaemonSet) (bool, error) {
		if dd.Status.ActiveReplicaSet == "" {
			return false, nil
		}
		if dd.Status.Available != 3 {
			return false, nil
		}

		return true, nil
	}
	err = utils.WaitForFuncOnExtendedDaemonset(t, f.Client, namespace, name, isOK, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}
}

func UseExtendedDaemonsetSetting(t *testing.T) {
	namespace, ctx, f := initTestFwkResources(t, "extendeddaemonset")
	defer ctx.Cleanup()
	edsName := "eds"

	resouresRef := corev1.ResourceList{
		"cpu":    resource.MustParse("0.1"),
		"memory": resource.MustParse("20M"),
	}
	edsNode := utils.NewExtendedDaemonsetSetting(namespace, "test-eds", edsName, &utils.NewExtendedDaemonsetSettingOptions{
		Selector: map[string]string{"overwrite": "test-eds"},
		Resources: map[string]corev1.ResourceRequirements{
			"main": {
				Requests: resouresRef,
			},
		},
	})
	err := f.Client.Create(goctx.TODO(), edsNode, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	nodeName := "kind-worker2"
	nodeWorker2 := &corev1.Node{}
	nodeKey := dynclient.ObjectKey{
		Name: nodeName,
	}
	err = f.Client.Get(goctx.TODO(), nodeKey, nodeWorker2)
	if err != nil {
		t.Fatal(err)
	}
	if nodeWorker2.Labels == nil {
		nodeWorker2.Labels = make(map[string]string)
	}
	nodeWorker2.Labels["overwrite"] = "test-eds"
	err = f.Client.Update(goctx.TODO(), nodeWorker2)

	daemonset := utils.NewExtendedDaemonset(namespace, edsName, fmt.Sprintf("k8s.gcr.io/pause:%s", "3.1"), nil)
	err = f.Client.Create(goctx.TODO(), daemonset, &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	isDSOK := func(ds *datadoghqv1alpha1.ExtendedDaemonSet) (bool, error) {
		if ds.Status.Ready == 3 {
			return true, nil
		}
		return false, nil
	}
	err = utils.WaitForFuncOnExtendedDaemonset(t, f.Client, namespace, edsName, isDSOK, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	podList := &corev1.PodList{}
	err = f.Client.List(goctx.TODO(), podList, &dynclient.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.Set(map[string]string{"extendeddaemonset.datadoghq.com/name": edsName}).AsSelector(),
	})
	var podOK bool
	for _, pod := range podList.Items {
		if pod.Spec.NodeName == nodeName {
			for _, container := range pod.Spec.Containers {
				if container.Name != "main" {
					continue
				}
				if diff := cmp.Diff(resouresRef, container.Resources.Requests); diff == "" {
					podOK = true
					break
				} else {
					t.Logf("diff pods resources: %s", diff)
				}
			}
		}
		if podOK {
			break
		}
	}
	if !podOK {
		t.Fatalf("unable to find updated pod")
	}

}

func initTestFwkResources(t *testing.T, deploymentName string) (string, *framework.TestCtx, *framework.Framework) {
	ctx := framework.NewTestCtx(t)
	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetOperatorNamespace()
	if err != nil {
		t.Fatal(err)
	}
	err = GenerateClusterRoleManifest(t, ctx, namespace, ctx.GetID(), deployDirPath)
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for extendeddaemonset-controller to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, deploymentName, 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	pods, err2 := f.KubeClient.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err2 != nil {
		t.Fatal(err2)
	}
	kubecontrolerPod := corev1.Pod{}
	kubecontrolerPod.Name = "kube-controller-manager-kind-control-plane"
	kubecontrolerPod.Namespace = "kube-system"

	pods.Items = append(pods.Items, kubecontrolerPod)
	options := &corev1.PodLogOptions{
		Follow: true,
	}
	for _, pod := range pods.Items {

		req := f.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, options)
		go func(t *testing.T, name string) {
			t.Logf("Add logger for pod:%s", name)
			readCloser, err := req.Stream()
			if err != nil {
				return
			}
			ctx.AddCleanupFn(
				func() error {
					_ = readCloser.Close()
					t.Logf("end reader [%s]", name)
					return nil
				})
			w := &logWriter{
				name: name,
				t:    t,
			}
			_, _ = io.Copy(w, readCloser)
		}(t, pod.Name)
	}

	return namespace, ctx, f
}

type logWriter struct {
	name string
	t    *testing.T
}

func (l *logWriter) Write(b []byte) (int, error) {
	l.t.Logf("#[%s] %s", l.name, string(b))
	return len(b), nil
}

// GenerateCombinedNamespacedManifest creates a temporary manifest yaml
// by combining all standard namespaced resource manifests in deployDir.
func GenerateClusterRoleManifest(t *testing.T, ctx *framework.TestCtx, namespace, id, deployDir string) error {
	saByte, err := ioutil.ReadFile(filepath.Join(deployDir, serviceAccountYamlFile))
	if err != nil {
		log.Warnf("Could not find the serviceaccount manifest: (%v)", err)
	}
	roleByte, err := ioutil.ReadFile(filepath.Join(deployDir, clusterRoleYamlFile))
	if err != nil {
		log.Warnf("Could not find role manifest: (%v)", err)
	}
	roleBindingByte, err := ioutil.ReadFile(filepath.Join(deployDir, clusterRoleBindingYamlFile))
	if err != nil {
		log.Warnf("Could not find role_binding manifest: (%v)", err)
	}

	var sa *corev1.ServiceAccount
	var clusterRole *rbacv1.ClusterRole
	var clusterRoleBinding *rbacv1.ClusterRoleBinding
	for _, fileByte := range [][]byte{saByte, roleByte, roleBindingByte} {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, _ := decode(fileByte, nil, nil)

		switch o := obj.(type) {
		case *corev1.ServiceAccount:
			sa = o
		case *rbacv1.ClusterRole:
			clusterRole = o
		case *rbacv1.ClusterRoleBinding:
			clusterRoleBinding = o
		default:
			fmt.Println("default case")
		}
	}

	clusterRole.Name = fmt.Sprintf("%s-%s", clusterRole.Name, id)
	clusterRoleBinding.Name = fmt.Sprintf("%s-%s", clusterRoleBinding.Name, id)
	{
		clusterRoleBinding.RoleRef.Name = clusterRole.Name

		for i, subject := range clusterRoleBinding.Subjects {
			if subject.Kind == "ServiceAccount" && subject.Name == sa.Name {
				clusterRoleBinding.Subjects[i].Namespace = namespace
			}
		}
	}
	t.Logf("ClusterRole: %#v", clusterRole)
	t.Logf("ClusterRoleBinding: %#v", clusterRoleBinding)
	cleanupOption := &framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval}

	if err = framework.Global.Client.Create(goctx.TODO(), clusterRole, cleanupOption); err != nil {
		return err
	}
	if err = framework.Global.Client.Create(goctx.TODO(), clusterRoleBinding, cleanupOption); err != nil {
		return err
	}

	return nil
}

const (
	deployDirPath              = "deploy"
	serviceAccountYamlFile     = "service_account.yaml"
	clusterRoleYamlFile        = "clusterrole.yaml"
	clusterRoleBindingYamlFile = "clusterrole_binding.yaml"
)
