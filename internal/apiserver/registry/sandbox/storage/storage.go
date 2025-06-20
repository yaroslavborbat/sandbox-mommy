package storage

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	genericreq "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	configrest "k8s.io/client-go/rest"

	corelisters "github.com/yaroslavborbat/sandbox-mommy/api/client/generated/listers/core/v1alpha1"
	subv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/subresources/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/registry/sandbox/client"
	sandboxrest "github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/registry/sandbox/rest"
)

type Storage struct {
	sandboxLister corelisters.SandboxLister
	groupResource schema.GroupResource
	attach        *sandboxrest.AttachREST
}

var (
	_ rest.KindProvider         = &Storage{}
	_ rest.Storage              = &Storage{}
	_ rest.Scoper               = &Storage{}
	_ rest.Getter               = &Storage{}
	_ rest.SingularNameProvider = &Storage{}
)

func NewStorage(
	serviceAccount types.NamespacedName,
	sandboxLister corelisters.SandboxLister,
	client client.GenericClient,
	restConfig *configrest.Config,
) *Storage {
	return &Storage{
		sandboxLister: sandboxLister,
		groupResource: subv1alpha1.Resource("sandbox"),
		attach:        sandboxrest.NewAttachREST(serviceAccount, sandboxLister, client, restConfig),
	}
}

func (s Storage) Kind() string {
	return "Sandbox"
}

func (s Storage) New() runtime.Object {
	return &subv1alpha1.Sandbox{}
}

func (s Storage) Destroy() {}

func (s Storage) NamespaceScoped() bool {
	return true
}

func (s Storage) Get(ctx context.Context, name string, _ *metav1.GetOptions) (runtime.Object, error) {
	namespace := genericreq.NamespaceValue(ctx)
	sandbox, err := s.sandboxLister.Sandboxes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &subv1alpha1.Sandbox{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Sandbox",
			APIVersion: subv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: sandbox.ObjectMeta,
	}, nil
}

func (s Storage) GetSingularName() string {
	return "sandbox"
}

func (s Storage) AttachREST() *sandboxrest.AttachREST {
	return s.attach
}
