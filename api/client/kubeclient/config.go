/*
Copyright 2018 The KubeVirt Authors
Copyright 2024 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Initially copied from https://github.com/kubevirt/kubevirt/blob/main/staging/src/kubevirt.io/client-go/kubecli/kubecli.go
*/

package kubeclient

import (
	"os"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/yaroslavborbat/sandbox-mommy/api/client/generated/clientset/versioned"
	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
)

func DefaultClientConfig(flags *pflag.FlagSet) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// use the standard defaults for this client command
	// DEPRECATED: remove and replace with something more accurate
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig

	flags.StringVar(&loadingRules.ExplicitPath, "kubeconfig", "",
		"Path to the kubeconfig file to use for CLI requests.")

	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}

	flagNames := clientcmd.RecommendedConfigOverrideFlags("")

	clientcmd.BindOverrideFlags(overrides, flags, flagNames)
	clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, overrides, os.Stdin)

	return clientConfig
}

func GetClientFromRESTConfig(config *rest.Config) (Client, error) {
	shallowCopy := rest.CopyConfig(config)
	shallowCopy.GroupVersion = &v1alpha1.SchemeGroupVersion
	shallowCopy.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: Codecs}
	shallowCopy.APIPath = "/apis"
	shallowCopy.ContentType = runtime.ContentTypeJSON
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	restClient, err := rest.RESTClientFor(shallowCopy)
	if err != nil {
		return nil, err
	}
	sandboxClient, err := versioned.NewForConfig(shallowCopy)
	if err != nil {
		return nil, err
	}
	return &client{
		config:        config,
		shallowCopy:   shallowCopy,
		restClient:    restClient,
		sandboxClient: sandboxClient,
	}, nil
}

var GetClientFromClientConfig = func(cmdConfig clientcmd.ClientConfig) (Client, error) {
	config, err := cmdConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return GetClientFromRESTConfig(config)
}
