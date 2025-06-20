package create

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/clientconfig"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/cmds/common"
	"github.com/yaroslavborbat/sandbox-mommy/internal/sandbox/template"
)

const (
	example = `  # Create sandbox
  {{ProgramName}} create my-sandbox
  # Create sandbox with dry-run
  {{ProgramName}} create --dry-run my-sandbox`
)

type create struct {
	template string
	ttl      time.Duration
	print    bool
}

func NewCreateSandboxCommand() *cobra.Command {
	c := &create{}

	cmd := &cobra.Command{
		Use:     "create [Name]",
		Short:   "Create sandbox",
		Example: example,
		Args:    cobra.ExactArgs(1),

		RunE: c.Run,
	}

	cmd.Flags().StringVarP(&c.template, "template", "t", "", "Template name")
	cmd.Flags().DurationVarP(&c.ttl, "ttl", "l", 1*time.Hour, "Sandbox TTL")
	cmd.Flags().BoolVarP(&c.print, "print", "p", false, "Print the created sandbox")
	common.SetDryRun(cmd.Flags())

	cmd.SetUsageTemplate(template.UsageTemplate())
	return cmd
}

func (c *create) Run(cmd *cobra.Command, args []string) error {
	if c.template == "" {
		return fmt.Errorf("--template is required")
	}

	name := args[0]
	client, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	sandbox := newSandbox(name, namespace, c.template, c.ttl)

	opts := metav1.CreateOptions{
		DryRun: common.GetDryRun(),
	}

	if common.IsDryRun() {
		cmd.Println("Dry run mode, no resources will be created.")
	}

	sandbox, err = client.Sandboxes(namespace).Create(cmd.Context(), sandbox, opts)
	if err != nil {
		return err
	}
	if c.print {
		bytes, err := yaml.Marshal(sandbox)
		if err != nil {
			return err
		}
		cmd.Println(string(bytes))
	}

	return nil
}

func newSandbox(name, namespace, template string, ttl time.Duration) *v1alpha1.Sandbox {
	return &v1alpha1.Sandbox{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.SandboxKind,
			APIVersion: v1alpha1.SchemeGroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.SandboxSpec{
			Template: template,
			TTL: &metav1.Duration{
				Duration: ttl,
			},
		},
	}
}
