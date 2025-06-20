package reconciler

import (
	"context"
	"fmt"
	"maps"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yaroslavborbat/sandbox-mommy/pkg/patch"
)

type MetaUpdater[T client.Object] interface {
	Update(ctx context.Context, oldObj, newObj T) error
}

func NewMetaUpdater[T client.Object](client client.Client) MetaUpdater[T] {
	return metaUpdater[T]{
		client: client,
	}
}

type metaUpdater[T client.Object] struct {
	client client.Client
}

func (m metaUpdater[T]) Update(ctx context.Context, oldObj, newObj T) error {
	metadataPatch := patch.NewJSONPatch()

	if !slices.Equal(oldObj.GetFinalizers(), newObj.GetFinalizers()) {
		metadataPatch.Append(m.jsonPatchOpsForFinalizers(oldObj.GetFinalizers(), newObj.GetFinalizers())...)
	}
	if !maps.Equal(oldObj.GetAnnotations(), newObj.GetAnnotations()) {
		metadataPatch.Append(m.jsonPatchOpsForAnnotations(oldObj.GetAnnotations(), newObj.GetAnnotations())...)
	}
	if !maps.Equal(oldObj.GetLabels(), newObj.GetLabels()) {
		metadataPatch.Append(m.jsonPatchOpsForLabels(oldObj.GetLabels(), newObj.GetLabels())...)
	}

	if metadataPatch.Len() == 0 {
		return nil
	}

	metadataPatchBytes, err := metadataPatch.Payload()
	if err != nil {
		return err
	}

	jsonPatch := client.RawPatch(types.JSONPatchType, metadataPatchBytes)
	if err = m.client.Patch(ctx, newObj, jsonPatch); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("error patching metadata (%s): %w", string(metadataPatchBytes), err)
	}

	return nil
}

func (m metaUpdater[T]) jsonPatchOpsForFinalizers(oldFinalizers, newFinalizers []string) []patch.JSONPatchOperation {
	return []patch.JSONPatchOperation{
		patch.WithTestOp("/metadata/finalizers", oldFinalizers),
		patch.WithReplaceOp("/metadata/finalizers", newFinalizers),
	}
}

func (m metaUpdater[T]) jsonPatchOpsForAnnotations(oldAnnotations, newAnnotations map[string]string) []patch.JSONPatchOperation {
	return []patch.JSONPatchOperation{
		patch.WithTestOp("/metadata/annotations", oldAnnotations),
		patch.WithReplaceOp("/metadata/annotations", newAnnotations),
	}
}

func (m metaUpdater[T]) jsonPatchOpsForLabels(oldLabels, newLabels map[string]string) []patch.JSONPatchOperation {
	return []patch.JSONPatchOperation{
		patch.WithTestOp("/metadata/labels", oldLabels),
		patch.WithReplaceOp("/metadata/labels", newLabels),
	}
}
