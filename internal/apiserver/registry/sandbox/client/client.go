package client

import (
	"fmt"

	"github.com/deckhouse/virtualization/api/client/kubeclient"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"kubevirt.io/client-go/kubecli"

	"github.com/yaroslavborbat/sandbox-mommy/internal/featuregate"
)

type GenericClient interface {
	Kubernetes() kubernetes.Interface
	Kubevirt() (kubecli.KubevirtClient, error)
	DVP() (kubeclient.Client, error)
}

func NewGenericClientFromConfig(restConfig *rest.Config) (GenericClient, error) {
	kube, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return client{}, err
	}
	kubevirt, err := kubecli.GetKubevirtClientFromRESTConfig(restConfig)
	if err != nil {
		return client{}, err
	}
	dvp, err := kubeclient.GetClientFromRESTConfig(restConfig)
	if err != nil {
		return client{}, err
	}

	return NewGenericClient(kube, kubevirt, dvp), nil
}

func NewGenericClient(kube kubernetes.Interface, kubevirt kubecli.KubevirtClient, dvp kubeclient.Client) GenericClient {
	return &client{
		kubernetes: kube,
		kubevirt:   kubevirt,
		dvp:        dvp,
	}
}

type client struct {
	kubernetes kubernetes.Interface
	kubevirt   kubecli.KubevirtClient
	dvp        kubeclient.Client
}

func (c client) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

func (c client) Kubevirt() (kubecli.KubevirtClient, error) {
	if !featuregate.Enabled(featuregate.Kubevirt) {
		return nil, fmt.Errorf("featuregate %s is not enabled", featuregate.Kubevirt)
	}
	return c.kubevirt, nil
}

func (c client) DVP() (kubeclient.Client, error) {
	if !featuregate.Enabled(featuregate.DVP) {
		return nil, fmt.Errorf("featuregate %s is not enabled", featuregate.DVP)
	}
	return c.dvp, nil
}
