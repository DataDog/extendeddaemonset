// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package plugin

import (
	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// ExtendedDaemonsetOptions provides information required to manage ExtendedDaemonset
type ExtendedDaemonsetOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

// NewExtendedDaemonsetOptions provides an instance of ExtendedDaemonsetOptions with default values
func NewExtendedDaemonsetOptions(streams genericclioptions.IOStreams) *ExtendedDaemonsetOptions {
	return &ExtendedDaemonsetOptions{
		configFlags: genericclioptions.NewConfigFlags(false),

		IOStreams: streams,
	}
}

// NewCmdExtendedDaemonset provides a cobra command wrapping ExtendedDaemonsetOptions
func NewCmdExtendedDaemonset(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewExtendedDaemonsetOptions(streams)

	cmd := &cobra.Command{
		Use: "ExtendedDaemonset [subcommand] [flags]",
	}

	cmd.AddCommand(NewCmdValidate(streams))
	cmd.AddCommand(NewCmdGet(streams))
	cmd.AddCommand(NewCmdGetERS(streams))

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for processing the command
func (o *ExtendedDaemonsetOptions) Complete(cmd *cobra.Command, args []string) error {
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *ExtendedDaemonsetOptions) Validate() error {
	return nil
}

// Run use to run the command
func (o *ExtendedDaemonsetOptions) Run() error {
	return nil
}
