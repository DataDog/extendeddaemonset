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

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
)

var (
	validateExample = `
	# validate a canary replicaset
	%[1]s validate foo
`
)

// ValidateOptions provides information required to manage ExtendedDaemonSet
type ValidateOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace             string
	userExtendedDaemonSetName string
}

// NewValidateOptions provides an instance of GetOptions with default values
func NewValidateOptions(streams genericclioptions.IOStreams) *ValidateOptions {
	return &ValidateOptions{
		configFlags: genericclioptions.NewConfigFlags(false),

		IOStreams: streams,
	}
}

// NewCmdValidate provides a cobra command wrapping ValidateOptions
func NewCmdValidate(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewValidateOptions(streams)

	cmd := &cobra.Command{
		Use:          "validate an ExtendedDaemonSet canary replicaset",
		Short:        "validate canary replicaset",
		Example:      fmt.Sprintf(validateExample, "kubectl"),
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
func (o *ValidateOptions) Complete(cmd *cobra.Command, args []string) error {
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
func (o *ValidateOptions) Validate() error {

	if len(o.args) < 1 {
		return fmt.Errorf("the extendeddaemonset name is required")
	}

	return nil
}

// Run use to run the command
func (o *ValidateOptions) Run() error {

	eds := &v1alpha1.ExtendedDaemonSet{}
	err := o.client.Get(context.TODO(), client.ObjectKey{Namespace: o.userNamespace, Name: o.userExtendedDaemonSetName}, eds)
	if err != nil && errors.IsNotFound(err) {
		return fmt.Errorf("ExtendedDaemonSet %s/%s not found", o.userNamespace, o.userExtendedDaemonSetName)
	} else if err != nil {
		return fmt.Errorf("unable to get ExtendedDaemonSet, err: %w", err)
	}

	if eds.Status.Canary == nil {
		return fmt.Errorf("the ExtendedDaemonset is not currently running a canary replicaset")
	}
	rsName := eds.Status.Canary.ReplicaSet
	newEds := eds.DeepCopy()
	if newEds.Annotations == nil {
		newEds.Annotations = make(map[string]string)
	}

	if name, ok := newEds.Annotations[v1alpha1.ExtendedDaemonSetCanaryValidAnnotationKey]; ok {
		if name == rsName {
			return fmt.Errorf("canary replicaset '%s' already validated", name)
		}
	}
	newEds.Annotations[v1alpha1.ExtendedDaemonSetCanaryValidAnnotationKey] = rsName
	if err = o.client.Update(context.TODO(), newEds); err != nil {
		return fmt.Errorf("unable to validate the canary replicaset, err: %v", err)
	}

	fmt.Fprintf(o.Out, "Canary replicaset '%s' was validated properly for extendeddaemonset %s/%s.\n", rsName, o.userNamespace, o.userExtendedDaemonSetName)

	return nil
}
