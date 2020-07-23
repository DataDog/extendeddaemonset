// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package plugin

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/spf13/cobra"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/datadog/extendeddaemonset/pkg/apis/datadoghq/v1alpha1"
)

var (
	failExample = `
    # Fail a canary deployment
    kubectl eds fail foo
`
)

// FailOptions provides information required to manage ExtendedDaemonSet
type FailOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace             string
	userExtendedDaemonSetName string
}

// NewFailOptions provides an instance of GetOptions with default values
func NewFailOptions(streams genericclioptions.IOStreams) *FailOptions {
	return &FailOptions{
		configFlags: genericclioptions.NewConfigFlags(false),

		IOStreams: streams,
	}
}

// NewCmdFail provides a cobra command wrapping FailOptions
func NewCmdFail(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewFailOptions(streams)

	cmd := &cobra.Command{
		Use:          "fail [ExtendedDaemonSet name]",
		Short:        "fail canary deployment",
		Example:      failExample,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for processing the command
func (o *FailOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args
	var err error

	clientConfig := o.configFlags.ToRawKubeConfigLoader()
	// Create the Client for Read/Write operations.
	o.client, err = NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("unable to instantiate client, err: %v", err)
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

// Validate ensures that all required arguments and flag values are provided
func (o *FailOptions) Validate() error {

	if len(o.args) < 1 {
		return fmt.Errorf("the extendeddaemonset name is required")
	}

	return nil
}

// Run use to run the command
func (o *FailOptions) Run() error {
	eds := &v1alpha1.ExtendedDaemonSet{}
	err := o.client.Get(context.TODO(), client.ObjectKey{Namespace: o.userNamespace, Name: o.userExtendedDaemonSetName}, eds)
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("ExtendedDaemonSet %s/%s not found", o.userNamespace, o.userExtendedDaemonSetName)
	} else if err != nil {
		return fmt.Errorf("unable to get ExtendedDaemonSet, err: %v", err)
	}

	if eds.Spec.Strategy.Canary == nil {
		return fmt.Errorf("the ExtendedDaemonset does not have a canary")
	}

	newEds := eds.DeepCopy()
	newEds.Spec.Strategy.Canary.Failed = true

	if err = o.client.Update(context.TODO(), newEds); err != nil {
		return fmt.Errorf("unable to fail ExtendedDaemonset deployment, err: %v", err)
	}

	fmt.Fprintf(o.Out, "ExtendedDaemonset '%s/%s' canary deployment set to failed\n", o.userNamespace, o.userExtendedDaemonSetName)

	return nil
}
