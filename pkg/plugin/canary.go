package plugin

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// NewCmdCanary provides a cobra command to control canary deployments.
func NewCmdCanary(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "canary [subcommand] [flags]",
		Short: "control ExtendedDaemonset canary deployment",
	}

	cmd.AddCommand(NewCmdValidate(streams))
	cmd.AddCommand(NewCmdPause(streams))
	cmd.AddCommand(NewCmdUnpause(streams))
	cmd.AddCommand(NewCmdFail(streams))
	cmd.AddCommand(NewCmdReset(streams))

	return cmd
}
