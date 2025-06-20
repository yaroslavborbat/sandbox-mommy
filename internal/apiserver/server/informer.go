package server

import (
	"fmt"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"kubevirt.io/client-go/kubecli"

	"github.com/yaroslavborbat/sandbox-mommy/api/client/generated/clientset/versioned"
	"github.com/yaroslavborbat/sandbox-mommy/api/client/generated/informers/externalversions"
)

const (
	defaultResync = 6 * time.Hour
)

func sandboxInformerFactory(rest *rest.Config) (externalversions.SharedInformerFactory, error) {
	client, err := versioned.NewForConfig(rest)
	if err != nil {
		return nil, fmt.Errorf("unable to construct client: %w", err)
	}
	return externalversions.NewSharedInformerFactory(client, defaultResync), nil
}

func vmiInformerFactory(rest *rest.Config) (informers.SharedInformerFactory, error) {
	client, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}
	return informers.NewSharedInformerFactory(client, defaultResync), nil
}
