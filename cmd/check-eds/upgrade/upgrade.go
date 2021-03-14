// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// Package upgrade contains upgrade plugin command logic.
package upgrade

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/pkg/plugin/common"
)

var upgradeExample = `
	# wait until the end of the extendeddaemonset foo upgrade
	%[1]s upgrade foo
`

// Options provides information required to manage canary.
type Options struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string

	client client.Client

	genericclioptions.IOStreams

	userNamespace             string
	userExtendedDaemonSetName string
	checkPeriod               time.Duration
	checkTimeout              time.Duration
}

// NewOptions provides an instance of Options with default values.
func NewOptions(streams genericclioptions.IOStreams) *Options {
	return &Options{
		configFlags: genericclioptions.NewConfigFlags(false),

		IOStreams:    streams,
		checkPeriod:  10 * time.Second,
		checkTimeout: 2 * time.Hour,
	}
}

// NewCmdUpgrade provides a cobra command wrapping Options.
func NewCmdUpgrade(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewOptions(streams)

	cmd := &cobra.Command{
		Use:          "upgrade [ExtendedDaemonSet name]",
		Short:        "wait until end of an ExtendedDaemonSet upgrade",
		Example:      fmt.Sprintf(upgradeExample, "kubectl"),
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

// Complete sets all information required for processing the command.
func (o *Options) Complete(cmd *cobra.Command, args []string) error {
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

// Validate ensures that all required arguments and flag values are provided.
func (o *Options) Validate() error {
	if o.userExtendedDaemonSetName == "" {
		return fmt.Errorf("the ExtendedDaemonset name needs to be provided")
	}

	return nil
}

// Run use to run the command.
func (o *Options) Run() error {
	o.printOutf("start checking deployment state")

	checkUpgradeDown := func() (bool, error) {
		eds := &v1alpha1.ExtendedDaemonSet{}
		err := o.client.Get(context.TODO(), client.ObjectKey{Namespace: o.userNamespace, Name: o.userExtendedDaemonSetName}, eds)
		if err != nil && errors.IsNotFound(err) {
			return false, fmt.Errorf("ExtendedDaemonSet %s/%s not found", o.userNamespace, o.userExtendedDaemonSetName)
		} else if err != nil {
			return false, fmt.Errorf("unable to get ExtendedDaemonSet, err: %w", err)
		}

		if eds.Status.Canary != nil {
			o.printOutf("canary running")

			return false, nil
		}
		if eds.Status.UpToDate < eds.Status.Current {
			o.printOutf("still upgrading nb pods: %d, nb updated pods: %d", eds.Status.Current, eds.Status.UpToDate)

			return false, nil
		}
		o.printOutf("upgrade is now finished")

		return true, nil
	}

	return wait.Poll(o.checkPeriod, o.checkTimeout, checkUpgradeDown)
}

func (o *Options) printOutf(format string, a ...interface{}) {
	args := []interface{}{time.Now().UTC().Format("2006-01-02T15:04:05.999Z"), o.userNamespace, o.userExtendedDaemonSetName}
	args = append(args, a...)
	_, _ = fmt.Fprintf(o.Out, "[%s] ExtendedDaemonset '%s/%s': "+format+"\n", args...)
}
