package delete

import (
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/clientconfig"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/cmds/common"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/template"
)

const (
	example = `  # Delete sandbox
  {{ProgramName}} delete my-sandbox
  # Delete sandbox with dry-run
  {{ProgramName}} delete --dry-run my-sandbox`
)

type delete struct{}

func NewDeleteSandboxCommand() *cobra.Command {
	d := &delete{}

	cmd := &cobra.Command{
		Use:     "delete [Name]",
		Short:   "Delete sandbox",
		Example: example,
		Args:    cobra.ExactArgs(1),

		RunE: d.Run,
	}

	common.SetDryRun(cmd.Flags())

	cmd.SetUsageTemplate(template.UsageTemplate())
	return cmd
}

func (d delete) Run(cmd *cobra.Command, args []string) error {
	name := args[0]
	client, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	opts := metav1.DeleteOptions{
		DryRun: common.GetDryRun(),
	}

	if common.IsDryRun() {
		cmd.Println("Dry run mode, no resources will be deleted.")
	}

	return client.Sandboxes(namespace).Delete(cmd.Context(), name, opts)
}
