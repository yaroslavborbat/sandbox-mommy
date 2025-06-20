package sandbox

import (
	"context"
	"log/slog"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/controller/service"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/controller/validator"
)

func NewValidator(log *slog.Logger) admission.CustomValidator {
	return validator.NewValidator[*v1alpha1.Sandbox](log.With("webhook", "validation")).
		WithCreateValidators(volumesValidator{}, typeValidator{})
}

type volumesValidator struct {
	validator service.VolumesValidator
}

func (v volumesValidator) ValidateCreate(ctx context.Context, sandbox *v1alpha1.Sandbox) (admission.Warnings, error) {
	if sandbox.Spec.TemplateSpec != nil {
		return v.validator.Validate(ctx, sandbox.Spec.TemplateSpec)
	}
	return admission.Warnings{}, nil
}

type typeValidator struct {
	validator service.TypeValidator
}

func (v typeValidator) ValidateCreate(ctx context.Context, sandbox *v1alpha1.Sandbox) (admission.Warnings, error) {
	if sandbox.Spec.TemplateSpec != nil {
		return v.validator.Validate(ctx, sandbox.Spec.TemplateSpec)
	}
	return admission.Warnings{}, nil
}
