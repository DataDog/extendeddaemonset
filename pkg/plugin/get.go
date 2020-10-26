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
	getExample = `
	# view all extendeddaemonset
	%[1]s get in the current namespace
	# view extendeddaemonset foo
	%[1]s get foo
`
)

// GetOptions provides information required to manage Kanary
type GetOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace             string
	userExtendedDaemonSetName string
}

// NewGetOptions provides an instance of GetOptions with default values
func NewGetOptions(streams genericclioptions.IOStreams) *GetOptions {
	return &GetOptions{
		configFlags: genericclioptions.NewConfigFlags(false),

		IOStreams: streams,
	}
}

// NewCmdGet provides a cobra command wrapping GetOptions
func NewCmdGet(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewGetOptions(streams)

	cmd := &cobra.Command{
		Use:          "get [ExtendedDaemonSet name]",
		Short:        "get ExtendedDaemonSet deployment(s)",
		Example:      fmt.Sprintf(getExample, "kubectl"),
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
func (o *GetOptions) Complete(cmd *cobra.Command, args []string) error {
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
func (o *GetOptions) Validate() error {

	if len(o.args) > 1 {
		return fmt.Errorf("either one or no arguments are allowed")
	}

	return nil
}

// Run use to run the command
func (o *GetOptions) Run() error {
	edsList := &v1alpha1.ExtendedDaemonSetList{}

	if o.userExtendedDaemonSetName == "" {
		err := o.client.List(context.TODO(), edsList, &client.ListOptions{Namespace: o.userNamespace})
		if err != nil {
			return fmt.Errorf("unable to list ExtendedDaemonSet, err: %v", err)
		}
	} else {
		eds := &v1alpha1.ExtendedDaemonSet{}
		err := o.client.Get(context.TODO(), client.ObjectKey{Namespace: o.userNamespace, Name: o.userExtendedDaemonSetName}, eds)
		if err != nil && errors.IsNotFound(err) {
			return fmt.Errorf("ExtendedDaemonSet %s/%s not found", o.userNamespace, o.userExtendedDaemonSetName)
		} else if err != nil {
			return fmt.Errorf("unable to get ExtendedDaemonSet, err: %w", err)
		}
		edsList.Items = append(edsList.Items, *eds)
	}

	table := newGetTable(o.Out)
	for _, item := range edsList.Items {
		data := []string{item.Namespace, item.Name, intToString(item.Status.Desired), intToString(item.Status.Current), intToString(item.Status.Ready), intToString(item.Status.UpToDate), intToString(item.Status.Available), intToString(item.Status.IgnoredUnresponsiveNodes), string(item.Status.State), string(item.Status.Reason), item.Status.ActiveReplicaSet, getCanaryRS(&item), getDuration(&item.ObjectMeta)}
		table.Append(data)
	}

	table.Render() // Send output

	return nil
}

func newGetTable(out io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(out)
	table.SetHeader([]string{"Namespace", "Name", "Desired", "Current", "Ready", "Up-to-date", "Available", "Ignored Unresponsive Nodes", "Status", "Reason", "Active RS", "Canary RS", "Age"})
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
