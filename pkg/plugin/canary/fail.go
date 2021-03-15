// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package canary

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/pkg/plugin/common"
)

const (
	cmdFail = true
)

var failExample = `
    # %[1]s a canary deployment
    kubectl eds %[1]s foo
`

// failOptions provides information required to manage ExtendedDaemonSet.
type failOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace             string
	userExtendedDaemonSetName string
	failStatus                bool
}

// newfailOptions provides an instance of GetOptions with default values.
func newfailOptions(streams genericclioptions.IOStreams, failStatus bool) *failOptions {
	return &failOptions{
		configFlags: genericclioptions.NewConfigFlags(false),

		IOStreams: streams,

		failStatus: failStatus,
	}
}

// newCmdFail provides a cobra command wrapping failOptions.
func newCmdFail(streams genericclioptions.IOStreams) *cobra.Command {
	o := newfailOptions(streams, cmdFail)

	cmd := &cobra.Command{
		Use:          "fail [ExtendedDaemonSet name]",
		Short:        "fail canary deployment",
		Example:      fmt.Sprintf(failExample, "fail"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.complete(c, args); err != nil {
				return err
			}
			if err := o.validate(); err != nil {
				return err
			}

			return o.run()
		},
	}

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// complete sets all information required for processing the command.
func (o *failOptions) complete(cmd *cobra.Command, args []string) error {
	o.args = args
	var err error

	clientConfig := o.configFlags.ToRawKubeConfigLoader()
	// Create the Client for Read/Write operations.
	o.client, err = common.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("unable to instantiate client, err: %w", err)
	}

	o.userNamespace, _, err = clientConfig.Namespace()
	if err != nil {
		return err
	}

	ns, err2 := cmd.Flags().GetString("namespace")
	if err2 != nil {
		return err
	}
	if ns != "" {
		o.userNamespace = ns
	}

	if len(args) > 0 {
		o.userExtendedDaemonSetName = args[0]
	}

	return nil
}

// validate ensures that all required arguments and flag values are provided.
func (o *failOptions) validate() error {
	if len(o.args) < 1 {
		return fmt.Errorf("the extendeddaemonset name is required")
	}

	return nil
}

// run use to run the command.
func (o *failOptions) run() error {
	eds := &v1alpha1.ExtendedDaemonSet{}
	err := o.client.Get(context.TODO(), client.ObjectKey{Namespace: o.userNamespace, Name: o.userExtendedDaemonSetName}, eds)
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("ExtendedDaemonSet %s/%s not found", o.userNamespace, o.userExtendedDaemonSetName)
	} else if err != nil {
		return fmt.Errorf("unable to get ExtendedDaemonSet, err: %w", err)
	}

	if eds.Spec.Strategy.Canary == nil {
		return fmt.Errorf("the ExtendedDaemonset does not have a canary strategy")
	}
	if eds.Status.Canary == nil {
		return fmt.Errorf("the ExtendedDaemonset does not have an active canary deployment")
	}

	newEds := eds.DeepCopy()
	if newEds.Annotations == nil {
		newEds.Annotations = make(map[string]string)
	} else if isFailed, ok := newEds.Annotations[v1alpha1.ExtendedDaemonSetCanaryFailedAnnotationKey]; ok {
		if o.failStatus && isFailed == v1alpha1.ValueStringTrue {
			return fmt.Errorf("canary deployment already failed")
		} else if !o.failStatus && isFailed == v1alpha1.ValueStringFalse {
			return fmt.Errorf("canary deployment already reset")
		}
	}
	newEds.Annotations[v1alpha1.ExtendedDaemonSetCanaryFailedAnnotationKey] = fmt.Sprintf("%v", o.failStatus)
	patch := client.MergeFrom(eds)
	if err = o.client.Patch(context.TODO(), newEds, patch); err != nil {
		return fmt.Errorf("unable to fail or reset ExtendedDaemonset deployment, err: %w", err)
	}
	action := "set to failed"
	if !o.failStatus {
		action = "reset"
	}
	fmt.Fprintf(o.Out, "ExtendedDaemonset '%s/%s' canary deployment %s\n", o.userNamespace, o.userExtendedDaemonSetName, action)

	return nil
}
