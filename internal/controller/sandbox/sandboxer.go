package sandbox

import (
	"context"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	sandboxcondition "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1/sandbox-condition"
)

type Sandboxer interface {
	Create(ctx context.Context, sandbox *v1alpha1.Sandbox, templateSpec *v1alpha1.SandboxTemplateSpec) error
	Delete(ctx context.Context, sandbox *v1alpha1.Sandbox) error
	Status(ctx context.Context, sandbox *v1alpha1.Sandbox) (metav1.ConditionStatus, sandboxcondition.Reason, string, error)
}

func NewSandboxer(sandboxType v1alpha1.SandboxType, client client.Client, log *slog.Logger) Sandboxer {
	switch sandboxType {
	case v1alpha1.SandboxTypePod:
		return NewPodSandboxer(client, log)
	case v1alpha1.SandboxTypeDVPVM:
		return NewDVPSandboxer(client, log)
	case v1alpha1.SandboxTypeKubevirtVMI:
		return NewKubevirtSandboxer(client, log)
	}
	return nil
}
