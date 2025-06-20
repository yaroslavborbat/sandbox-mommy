package server

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/rest"

	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/api"
	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/registry/sandbox/client"
)

var ErrConfigInvalid = errors.New("configuration is invalid")

type Config struct {
	Apiserver      *genericapiserver.Config
	Rest           *rest.Config
	ServiceAccount types.NamespacedName
}

func (c Config) Validate() error {
	if c.Apiserver == nil {
		return fmt.Errorf("%w: %s", ErrConfigInvalid, "Apiserver is required")
	}
	if c.Rest == nil {
		return fmt.Errorf("%w: %s", ErrConfigInvalid, "Rest is required")
	}
	if c.ServiceAccount.Name == "" || c.ServiceAccount.Namespace == "" {
		return fmt.Errorf("%w: %s", ErrConfigInvalid, "ServiceAccount is required")
	}
	return nil
}

func (c Config) Complete() (*Server, error) {
	sandboxSharedInformerFactory, err := sandboxInformerFactory(c.Rest)
	if err != nil {
		return nil, err
	}
	sandboxInformer := sandboxSharedInformerFactory.Sandbox().V1alpha1().Sandboxes()

	genericServer, err := c.Apiserver.Complete(nil).New("sandbox-api", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	genericClient, err := client.NewGenericClientFromConfig(c.Rest)
	if err != nil {
		return nil, err
	}

	if err = api.Install(
		sandboxInformer.Lister(),
		genericServer,
		genericClient,
		c.ServiceAccount,
		c.Rest,
	); err != nil {
		return nil, err
	}

	return NewServer(
		sandboxInformer.Informer(),
		genericServer,
	), nil
}
