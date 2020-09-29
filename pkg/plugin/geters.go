// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package plugin

import (
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/olekukonko/tablewriter"

	"github.com/spf13/cobra"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
)

var (
	getErsExample = `
	# view all extendeddaemonsetreplicaset
	%[1]s get-ers in the current namespace
	# view extendeddaemonsetreplicaset foo-dsfsfs
	%[1]s get-ers foo-dsfsfs
`
)

// GetERSOptions provides information required to manage Kanary
type GetERSOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace                       string
	userExtendedDaemonSetReplicaSetName string
}

// NewGetERSOptions provides an instance of GetERSOptions with default values
func NewGetERSOptions(streams genericclioptions.IOStreams) *GetERSOptions {
	return &GetERSOptions{
		configFlags: genericclioptions.NewConfigFlags(false),

		IOStreams: streams,
	}
}

// NewCmdGetERS provides a cobra command wrapping GetERSOptions
func NewCmdGetERS(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewGetERSOptions(streams)

	cmd := &cobra.Command{
		Use:          "get-ers [ExtendedDaemonSetReplicaset name]",
		Short:        "get-ers ExtendedDaemonSetReplicaset deployment(s)",
		Example:      fmt.Sprintf(getErsExample, "kubectl"),
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
func (o *GetERSOptions) Complete(cmd *cobra.Command, args []string) error {
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
		o.userExtendedDaemonSetReplicaSetName = args[0]
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *GetERSOptions) Validate() error {

	if len(o.args) > 1 {
		return fmt.Errorf("either one or no arguments are allowed")
	}

	return nil
}

// Run use to run the command
func (o *GetERSOptions) Run() error {
	ersList := &v1alpha1.ExtendedDaemonSetReplicaSetList{}

	if o.userExtendedDaemonSetReplicaSetName == "" {
		err := o.client.List(context.TODO(), ersList, &client.ListOptions{Namespace: o.userNamespace})
		if err != nil {
			return fmt.Errorf("unable to list ExtendedDaemonSetReplicaset, err: %v", err)
		}
	} else {
		ers := &v1alpha1.ExtendedDaemonSetReplicaSet{}
		err := o.client.Get(context.TODO(), client.ObjectKey{Namespace: o.userNamespace, Name: o.userExtendedDaemonSetReplicaSetName}, ers)
		if err != nil && errors.IsNotFound(err) {
			return fmt.Errorf("ExtendedDaemonSet %s/%s not found", o.userNamespace, o.userExtendedDaemonSetReplicaSetName)
		} else if err != nil {
			return fmt.Errorf("unable to get ExtendedDaemonSetReplicaset, err: %v", err)
		}
		ersList.Items = append(ersList.Items, *ers)
	}

	table := newGetERSTable(o.Out)
	for _, item := range ersList.Items {
		data := []string{item.Namespace, item.Name, intToString(item.Status.Desired), intToString(item.Status.Current), intToString(item.Status.Ready), intToString(item.Status.Available), intToString(item.Status.IgnoredUnresponsiveNodes), item.Status.Status, getDuration(&item.ObjectMeta)}
		table.Append(data)
	}

	table.Render() // Send output

	return nil
}

func newGetERSTable(out io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Namespace", "Name", "Desired", "Current", "Ready", "Available", "Ignored Unresponsive Nodes", "Status", "Age"})
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)

	return table
}
