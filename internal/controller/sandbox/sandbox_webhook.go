package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func NewDefaulter(log *slog.Logger) admission.CustomDefaulter {
	return Defaulter{
		log: log.With("webhook", "defaulter"),
	}
}

type Defaulter struct {
	log *slog.Logger
}

func (d Defaulter) Default(_ context.Context, obj runtime.Object) error {
	sandbox, ok := obj.(*v1alpha1.Sandbox)
	if !ok {
		d.log.Error(fmt.Sprintf("Expected a Sandbox but got a %T", obj))
		return nil
	}
	if sandbox.Spec.TTL.Duration == 0 {
		sandbox.Spec.TTL = metav1.Duration{
			Duration: time.Hour * 1,
		}
	}
	return nil
}
