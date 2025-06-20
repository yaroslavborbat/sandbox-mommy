package sandbox

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
	controllerName  = "sandbox-controller"
	labelSandboxUID = "sandbox.io/uid"
)

func SetupController(mgr ctrl.Manager, log *slog.Logger) error {
	log = log.With(logging.SlogController(controllerName))
	c := mgr.GetClient()
	r := reconciler.NewBaseReconciler(v1alpha1.SandboxKind, c,
		func() *v1alpha1.Sandbox {
			return &v1alpha1.Sandbox{}
		},
		reconciler.NewStatusUpdater[*v1alpha1.Sandbox](c, func(obj *v1alpha1.Sandbox) interface{} {
			return obj.Status
		}),
		reconciler.NewMetaUpdater[*v1alpha1.Sandbox](c),
		NewReconciler(c, mgr.GetEventRecorderFor(controllerName), NewSandboxer))
	if err := r.SetupWithManager(mgr, log); err != nil {
		return fmt.Errorf("failed to setup %q: %w", controllerName, err)
	}

	if err := builder.WebhookManagedBy(mgr).
		For(&v1alpha1.Sandbox{}).
		WithValidator(NewValidator(log)).
		WithDefaulter(NewDefaulter(log)).
		Complete(); err != nil {
		return err
	}

	log.Info("Registered sandbox controller")
	return nil
}
