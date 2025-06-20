package reconciler

import (
	"context"

	"k8s.io/apimachinery/pkg/api/equality"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusUpdater[T client.Object] interface {
	Update(ctx context.Context, objOld, objNew T) error
}

func NewStatusUpdater[T client.Object](client client.Client, getStatus func(obj T) interface{}) StatusUpdater[T] {
	return statusUpdater[T]{
		client:    client,
		getStatus: getStatus,
	}
}

type statusUpdater[T client.Object] struct {
	client    client.Client
	getStatus func(obj T) interface{}
}

func (s statusUpdater[T]) Update(ctx context.Context, objOld, objNew T) error {
	if equality.Semantic.DeepEqual(s.getStatus(objOld), s.getStatus(objNew)) {
		return nil
	}
	return s.client.Status().Update(ctx, objNew)
}
