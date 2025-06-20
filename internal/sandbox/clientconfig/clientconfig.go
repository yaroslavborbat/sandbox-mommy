package clientconfig

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/yaroslavborbat/sandbox-mommy/api/client/kubeclient"
)

type key struct{}

var clientConfigKey key

// NewContext returns a new Context that stores a clientConfig as value.
func NewContext(ctx context.Context, clientConfig clientcmd.ClientConfig) context.Context {
	return context.WithValue(ctx, clientConfigKey, clientConfig)
}

func ClientAndNamespaceFromContext(ctx context.Context) (client kubeclient.Client, namespace string, overridden bool, err error) {
	clientConfig, ok := ctx.Value(clientConfigKey).(clientcmd.ClientConfig)
	if !ok {
		return nil, "", false, fmt.Errorf("unable to get client config from context")
	}
	client, err = kubeclient.GetClientFromClientConfig(clientConfig)
	if err != nil {
		return nil, "", false, err
	}
	namespace, overridden, err = clientConfig.Namespace()
	if err != nil {
		return nil, "", false, err
	}
	return client, namespace, overridden, nil
}
