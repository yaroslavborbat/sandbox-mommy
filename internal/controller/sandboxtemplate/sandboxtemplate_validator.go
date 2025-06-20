package sandboxtemplate

import (
	"context"
	"log/slog"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/controller/service"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/controller/validator"
)

func NewValidator(log *slog.Logger) admission.CustomValidator {
	return validator.NewValidator[*v1alpha1.SandboxTemplate](log.With("webhook", "validation")).
		WithCreateValidators(volumesValidator{}, typeValidator{})
}

type volumesValidator struct {
	validator service.VolumesValidator
}

func (v volumesValidator) ValidateCreate(ctx context.Context, template *v1alpha1.SandboxTemplate) (admission.Warnings, error) {
	return v.validator.Validate(ctx, &template.Spec)
}

type typeValidator struct {
	validator service.TypeValidator
}

func (v typeValidator) ValidateCreate(ctx context.Context, template *v1alpha1.SandboxTemplate) (admission.Warnings, error) {
	return v.validator.Validate(ctx, &template.Spec)
}
