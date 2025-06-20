package reconciler

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

type Reconciler[T client.Object] interface {
	Reconcile(ctx context.Context, obj T) (reconcile.Result, error)
	Setup(reconciler reconcile.Reconciler, mgr ctrl.Manager, log *slog.Logger) error
}

type NewObject[T client.Object] func() T

func NewBaseReconciler[T client.Object](
	kind string,
	client client.Client,
	newObject NewObject[T],
	statusUpdater StatusUpdater[T],
	metaUpdater MetaUpdater[T],
	reconciler Reconciler[T],
) *BaseReconciler[T] {
	return &BaseReconciler[T]{
		kind:          kind,
		client:        client,
		newObject:     newObject,
		statusUpdater: statusUpdater,
		metaUpdater:   metaUpdater,
		reconciler:    reconciler,
	}
}

type BaseReconciler[T client.Object] struct {
	kind          string
	client        client.Client
	newObject     NewObject[T]
	statusUpdater StatusUpdater[T]
	metaUpdater   MetaUpdater[T]
	reconciler    Reconciler[T]
}

func (b *BaseReconciler[T]) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := logging.FromContext(ctx).With(slog.String("kind", b.kind))
	log.Debug("Start reconciliation")
	defer func() {
		log.Debug("Reconcile finished")
	}()

	obj := b.newObject()
	err := b.client.Get(ctx, req.NamespacedName, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Object not found")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if reflect.ValueOf(obj).IsNil() {
		return reconcile.Result{}, nil
	}
	objCopy := obj.DeepCopyObject().(T)

	result, reconcileErr := b.reconciler.Reconcile(ctx, objCopy)
	if reconcileErr != nil {
		if apierrors.IsConflict(reconcileErr) {
			log.Debug("Conflict", logging.SlogErr(reconcileErr))
		} else {
			log.Error("Failed to reconcile", logging.SlogErr(reconcileErr))
		}
	}

	if err = b.update(ctx, obj, objCopy); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update resource: %w", err)
	}

	if reconcileErr != nil {
		return reconcile.Result{}, reconcileErr
	}

	return result, nil
}

func (b *BaseReconciler[T]) SetupWithManager(mgr ctrl.Manager, log *slog.Logger) error {
	return b.reconciler.Setup(b, mgr, log)
}

func (b *BaseReconciler[T]) update(ctx context.Context, oldObj, newObj T) error {
	if err := b.statusUpdater.Update(ctx, oldObj, newObj); err != nil {
		return err
	}

	return b.metaUpdater.Update(ctx, oldObj, newObj)
}
