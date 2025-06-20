package sandbox

import (
	"fmt"
	"log/slog"
	"time"

	dvpcorev1alpha2 "github.com/deckhouse/virtualization/api/core/v1alpha2"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	virtv1 "kubevirt.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	sandboxcondition "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1/sandbox-condition"
	"github.com/yaroslavborbat/sandbox-mommy/internal/common"
	"github.com/yaroslavborbat/sandbox-mommy/internal/featuregate"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/controller/condition"
	scontrollerutil "github.com/yaroslavborbat/sandbox-mommy/pkg/controller/util"

	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

func NewReconciler(client client.Client, recorder record.EventRecorder, managerCreator SandboxerCreator) *Reconciler {
	return &Reconciler{
		client:         client,
		recorder:       recorder,
		managerCreator: managerCreator,
	}
}

type Reconciler struct {
	client         client.Client
	recorder       record.EventRecorder
	managerCreator SandboxerCreator
}

type SandboxerCreator func(sandboxType v1alpha1.SandboxType, client client.Client, log *slog.Logger) Sandboxer

func (r *Reconciler) Reconcile(ctx context.Context, sandbox *v1alpha1.Sandbox) (reconcile.Result, error) {
	if sandbox == nil {
		return reconcile.Result{}, nil
	}

	log := logging.FromContext(ctx)

	cb := condition.NewConditionBuilder(sandboxcondition.TypeReady)
	cb.Generation(sandbox.Generation).
		Status(metav1.ConditionFalse).
		Reason(sandboxcondition.ReasonPending)

	sandboxTemplate, sandboxTemplateSpec, templateTerminating, err := r.handleTemplateSpec(ctx, sandbox, cb, log)
	if err != nil {
		return reconcile.Result{}, err
	}

	if sandbox.Status.Type == "" {
		sandbox.Status.Type = common.DetectSandboxType(sandboxTemplateSpec)
	}

	sandboxer := NewSandboxer(sandbox.Status.Type, r.client, log)

	if !sandbox.GetDeletionTimestamp().IsZero() {
		cb.
			Status(metav1.ConditionFalse).
			Reason(sandboxcondition.ReasonTerminating).
			Message("")
		condition.SetCondition(cb, &sandbox.Status.Conditions)
		return reconcile.Result{}, r.handleTerminating(ctx, sandbox, sandboxTemplate, sandboxer, log)
	}

	if isTTLExpired(sandbox) {
		log.Info("Sandbox is expired, deleting...")
		return reconcile.Result{}, r.client.Delete(ctx, sandbox)
	}

	if templateTerminating || sandboxTemplateSpec == nil || sandboxer == nil {
		return reconcile.Result{}, nil
	}

	controllerutil.AddFinalizer(sandbox, v1alpha1.FinalizerProtectBySandboxController)
	if err := scontrollerutil.ProtectObjects(ctx, r.client, v1alpha1.FinalizerProtectBySandboxController, sandboxTemplate); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to protect sandbox template: %w", err)
	}

	if err := sandboxer.Create(ctx, sandbox, sandboxTemplateSpec); err != nil {
		log.Error("Failed to create sandbox", logging.SlogErr(err))
		cb.
			Status(metav1.ConditionFalse).
			Reason(sandboxcondition.ReasonFailed).
			Message("Failed to create sandbox.")
		condition.SetCondition(cb, &sandbox.Status.Conditions)
		r.recorder.Eventf(sandbox, corev1.EventTypeWarning, sandboxcondition.ReasonFailed.String(), "Failed to create sandbox %v", err)
		return reconcile.Result{}, err
	}

	status, reason, message, err := sandboxer.Status(ctx, sandbox)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get sandbox status: %w", err)
	}
	cb.
		Status(status).
		Reason(reason).
		Message(message)
	condition.SetCondition(cb, &sandbox.Status.Conditions)

	return reconcile.Result{RequeueAfter: nextSync(sandbox)}, nil
}

func (r *Reconciler) handleTemplateSpec(ctx context.Context, sandbox *v1alpha1.Sandbox, cb *condition.ConditionBuilder, log *slog.Logger) (*v1alpha1.SandboxTemplate, *v1alpha1.SandboxTemplateSpec, bool, error) {
	if sandbox.Spec.TemplateSpec != nil {
		return nil, sandbox.Spec.TemplateSpec, false, nil
	}

	var (
		sandboxTemplate     *v1alpha1.SandboxTemplate
		sandboxTemplateSpec *v1alpha1.SandboxTemplateSpec
		templateTerminating bool
	)

	if sandbox.Spec.Template != "" {
		sandboxTmpl := &v1alpha1.SandboxTemplate{}
		err := r.client.Get(ctx, types.NamespacedName{Name: sandbox.Spec.Template}, sandboxTmpl)

		switch {
		case err != nil:
			if !apierrors.IsNotFound(err) {
				return nil, nil, false, err
			}
			log.Info("Sandbox template not found, waiting...")
			cb.
				Status(metav1.ConditionFalse).
				Reason(sandboxcondition.ReasonPending).
				Message(fmt.Sprintf("SandboxTemplate %q not found", sandboxTmpl.Name))
			condition.SetCondition(cb, &sandbox.Status.Conditions)
		case !sandboxTmpl.GetDeletionTimestamp().IsZero():
			cb.
				Status(metav1.ConditionFalse).
				Reason(sandboxcondition.ReasonTerminating).
				Message(fmt.Sprintf("SandboxTemplate %q is terminating, rejected for use.", sandboxTmpl.Name))
			condition.SetCondition(cb, &sandbox.Status.Conditions)
			sandboxTemplate = sandboxTmpl
			sandboxTemplateSpec = &sandboxTmpl.Spec
			templateTerminating = true
		default:
			sandboxTemplate = sandboxTmpl
			sandboxTemplateSpec = &sandboxTmpl.Spec
		}
	}

	return sandboxTemplate, sandboxTemplateSpec, templateTerminating, nil
}

func (r *Reconciler) handleTerminating(ctx context.Context, sandbox *v1alpha1.Sandbox, sandboxTemplate *v1alpha1.SandboxTemplate, sandboxer Sandboxer, log *slog.Logger) error {
	if sandbox == nil {
		return nil
	}

	log.Info("Sandbox is being deleted...")

	if sandboxer != nil {
		if err := sandboxer.Delete(ctx, sandbox); err != nil {
			return fmt.Errorf("failed to delete sandbox: %w", err)
		}
	} else {
		log.Warn("Cannot detect sandbox type, that's why some child resources can be deleted in background")
	}

	if sandboxTemplate != nil {
		if err := scontrollerutil.UnprotectObject(ctx, r.client, v1alpha1.FinalizerProtectBySandboxController, sandboxTemplate); err != nil {
			return fmt.Errorf("failed to unprotect sandbox template: %w", err)
		}
	}

	controllerutil.RemoveFinalizer(sandbox, v1alpha1.FinalizerProtectBySandboxController)
	return nil
}

func (r *Reconciler) Setup(reconciler reconcile.Reconciler, mgr ctrl.Manager, log *slog.Logger) error {
	b := ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&v1alpha1.Sandbox{}).
		Owns(&corev1.Pod{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldPod := e.ObjectOld.(*corev1.Pod)
				newPod := e.ObjectNew.(*corev1.Pod)

				return oldPod.Status.Phase != newPod.Status.Phase
			},
		})).
		WithOptions(controller.Options{
			RecoverPanic:   ptr.To(true),
			LogConstructor: logging.NewConstructor(log),
		})
	if featuregate.Enabled(featuregate.Kubevirt) {
		b = b.
			Owns(&virtv1.VirtualMachineInstance{}, builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldVMI := e.ObjectOld.(*virtv1.VirtualMachineInstance)
					newVMI := e.ObjectNew.(*virtv1.VirtualMachineInstance)

					return oldVMI.Status.Phase != newVMI.Status.Phase
				},
			}))
	}
	if featuregate.Enabled(featuregate.DVP) {
		b = b.
			Owns(&dvpcorev1alpha2.VirtualMachine{}, builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldVM := e.ObjectOld.(*dvpcorev1alpha2.VirtualMachine)
					newVM := e.ObjectNew.(*dvpcorev1alpha2.VirtualMachine)

					return oldVM.Status.Phase != newVM.Status.Phase
				},
			}))
	}
	return b.Complete(reconciler)
}

func isTTLExpired(sandbox *v1alpha1.Sandbox) bool {
	return time.Now().After(sandbox.GetCreationTimestamp().Add(sandbox.Spec.TTL.Duration))
}

func nextSync(sandbox *v1alpha1.Sandbox) time.Duration {
	if sandbox.Spec.TTL.Duration == 0 {
		return 0
	}
	expirationTime := sandbox.GetCreationTimestamp().Add(sandbox.Spec.TTL.Duration)
	return time.Until(expirationTime)
}
