package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/component-base/term"
	"k8s.io/component-base/version"
)

func NewSandboxAPIServerCommand() *cobra.Command {
	opts := NewOptions()
	cmd := &cobra.Command{
		Short: "Launch sandbox-api server",
		Long:  "Launch sandbox-api server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runCommand(cmd, opts); err != nil {
				return err
			}
			return nil
		},
	}
	fs := cmd.Flags()
	nfs := opts.Flags()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}
	local := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	logs.AddGoFlags(local)
	nfs.FlagSet("logging").AddGoFlagSet(local)

	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		if err != nil {
			return err
		}
		cliflag.PrintSections(cmd.OutOrStderr(), nfs, cols)
		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		if err != nil {
			panic(err)
		}
		cliflag.PrintSections(cmd.OutOrStdout(), nfs, cols)
	})
	fs.AddGoFlagSet(local)
	return cmd
}

func runCommand(cmd *cobra.Command, o *Options) error {
	if o.ShowVersion {
		cmd.Println(version.Get().GitVersion)
		return nil
	}

	err := o.Validate()
	if err != nil {
		return err
	}

	config, err := o.ServerConfig()
	if err != nil {
		return err
	}

	s, err := config.Complete()
	if err != nil {
		return err
	}

	return s.Run(cmd.Context())
}
