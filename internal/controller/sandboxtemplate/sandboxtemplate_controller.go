package sandboxtemplate

import (
	"fmt"
	"log/slog"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/controller/reconciler"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

const (
	controllerName = "sandboxtemplate-controller"
)

func SetupController(mgr ctrl.Manager, log *slog.Logger) error {
	log = log.With(logging.SlogController(controllerName))

	c := mgr.GetClient()
	r := reconciler.NewBaseReconciler(
		v1alpha1.SandboxTemplateKind,
		c,
		func() *v1alpha1.SandboxTemplate {
			return &v1alpha1.SandboxTemplate{}
		},
		reconciler.NewStatusUpdater[*v1alpha1.SandboxTemplate](c, func(obj *v1alpha1.SandboxTemplate) interface{} {
			return obj.Status
		}),
		reconciler.NewMetaUpdater[*v1alpha1.SandboxTemplate](c),
		NewReconciler(c))

	if err := r.SetupWithManager(mgr, log); err != nil {
		return fmt.Errorf("failed to setup %q: %w", controllerName, err)
	}

	if err := builder.WebhookManagedBy(mgr).
		For(&v1alpha1.SandboxTemplate{}).
		WithValidator(NewValidator(log)).
		Complete(); err != nil {
		return err
	}

	log.Info("Registered sandboxtemplate controller")
	return nil

}
