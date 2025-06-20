package util

import (
	"context"
	"reflect"
	"slices"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/yaroslavborbat/sandbox-mommy/pkg/patch"
)

func ProtectObjects(ctx context.Context, c client.Client, finalizer string, objs ...client.Object) error {
	for _, obj := range objs {
		if obj == nil || reflect.ValueOf(obj).IsNil() {
			continue
		}
		if err := ProtectObject(ctx, c, finalizer, obj); err != nil {
			return err
		}
	}
	return nil
}

func ProtectObject(ctx context.Context, c client.Client, finalizer string, obj client.Object) error {
	if controllerutil.ContainsFinalizer(obj, finalizer) {
		return nil
	}

	oldFinalizers := obj.GetFinalizers()
	newFinalizers := slices.Clone(oldFinalizers)
	newFinalizers = append(newFinalizers, finalizer)

	patchBytes, err := patch.NewJSONPatch(
		patch.WithTestOp("/metadata/finalizers", oldFinalizers),
		patch.WithReplaceOp("/metadata/finalizers", newFinalizers),
	).Payload()
	if err != nil {
		return err
	}

	return c.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, patchBytes))
}

func UnprotectObjects(ctx context.Context, c client.Client, finalizer string, objs ...client.Object) error {
	for _, obj := range objs {
		if obj == nil || reflect.ValueOf(obj).IsNil() {
			continue
		}

		if err := UnprotectObject(ctx, c, finalizer, obj); err != nil {
			return err
		}
	}
	return nil
}

func UnprotectObject(ctx context.Context, c client.Client, finalizer string, obj client.Object) error {
	containsFinalizer := false
	oldFinalizers := obj.GetFinalizers()
	var newFinalizers []string
	for _, s := range oldFinalizers {
		if s != finalizer {
			newFinalizers = append(newFinalizers, s)
		}
		containsFinalizer = true
	}
	if !containsFinalizer {
		return nil
	}

	patchBytes, err := patch.NewJSONPatch(
		patch.WithTestOp("/metadata/finalizers", oldFinalizers),
		patch.WithReplaceOp("/metadata/finalizers", newFinalizers),
	).Payload()
	if err != nil {
		return err
	}

	return c.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, patchBytes))
}
