package sandboxtemplate

import (
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"golang.org/x/net/context"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	sandboxtemplatecondition "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1/sandboxtemplate-condition"
	"github.com/yaroslavborbat/sandbox-mommy/internal/common"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/controller/condition"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/controller/reconciler"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

func NewReconciler(client client.Client) *Reconciler {
	return &Reconciler{
		client: client,
	}

}

var _ reconciler.Reconciler[*v1alpha1.SandboxTemplate] = &Reconciler{}

type Reconciler struct {
	client client.Client
}

func (r *Reconciler) Reconcile(_ context.Context, sandboxTemplate *v1alpha1.SandboxTemplate) (reconcile.Result, error) {
	if sandboxTemplate == nil {
		return reconcile.Result{}, nil
	}

	sandboxTemplate.Status.Type = common.DetectSandboxType(&sandboxTemplate.Spec)
	cb := condition.NewConditionBuilder(sandboxtemplatecondition.TypeReady)
	defer func() {
		condition.SetCondition(cb, &sandboxTemplate.Status.Conditions)
	}()
	cb.Generation(sandboxTemplate.Generation).
		Status(metav1.ConditionTrue).
		Reason(sandboxtemplatecondition.ReasonReady)

	if !sandboxTemplate.GetDeletionTimestamp().IsZero() {
		cb.Status(metav1.ConditionFalse).Reason(sandboxtemplatecondition.ReasonTerminating)
		if controllerutil.ContainsFinalizer(sandboxTemplate, v1alpha1.FinalizerProtectBySandboxController) {
			cb.Message("SandboxTemplate protected by sandbox-controller. Will be deleted after all sandboxes, which use this template terminated.")
		}
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) Setup(reconciler reconcile.Reconciler, mgr ctrl.Manager, log *slog.Logger) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&v1alpha1.SandboxTemplate{}).
		WithOptions(controller.Options{
			RecoverPanic:   ptr.To(true),
			LogConstructor: logging.NewConstructor(log),
		}).
		Complete(reconciler)
}
