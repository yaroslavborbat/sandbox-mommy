package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/yaroslavborbat/sandbox-mommy/api/client/kubeclient"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/clientconfig"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/cmds/attach"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/cmds/create"
	cmddelete "github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/cmds/delete"
)

const (
	long = `
 ____                  _ _
/ ___|  __ _ _ __   __| | |__   _____  __
\___ \ / _' | '_ \ / _' | '_ \ / _ \ \/ /
___) | (_| | | | | (_| | |_) | (_) >  <
|____/ \__'_|_| |_|\__'_|_'__/ \___/_/\_\

  Manage sandboxes
`
)

func NewSandboxCommand() *cobra.Command {
	cobra.AddTemplateFunc(
		"ProgramName", func() string {
			return programName
		},
	)

	cobra.AddTemplateFunc(
		"prepare", func(s string) string {
			result := strings.Replace(s, "{{ProgramName}}", programName, -1)
			return result
		},
	)

	rootCmd := &cobra.Command{
		Use:           fmt.Sprintf("%s [command]", programName),
		Short:         "Manage sandboxes",
		Long:          long,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.SetOut(os.Stdout)
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	rootCmd.SetContext(clientconfig.NewContext(
		ctx, kubeclient.DefaultClientConfig(rootCmd.PersistentFlags()),
	))

	rootCmd.AddCommand(
		create.NewCreateSandboxCommand(),
		cmddelete.NewDeleteSandboxCommand(),
		attach.NewAttachSandboxCommand(),
	)

	return rootCmd
}

var programName = getProgram(filepath.Base(os.Args[0]))

func getProgram(program string) string {
	if strings.HasPrefix(program, "kubectl-") {
		return fmt.Sprintf("kubectl %s", strings.TrimPrefix(program, "kubectl-"))
	}
	return program
}
