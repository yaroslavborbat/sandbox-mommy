package rest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	dvpcorev1alpha2 "github.com/deckhouse/virtualization/api/core/v1alpha2"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/proxy"
	genericreq "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	configrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
	virtv1 "kubevirt.io/api/core/v1"

	dvpkubeclient "github.com/deckhouse/virtualization/api/client/kubeclient"
	dvpsubs "github.com/deckhouse/virtualization/api/subresources/v1alpha2"

	corelisters "github.com/yaroslavborbat/sandbox-mommy/api/client/generated/listers/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	sanboxcondition "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1/sandbox-condition"
	subv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/subresources/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/registry/sandbox/client"
	"github.com/yaroslavborbat/sandbox-mommy/internal/common"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/controller/condition"
	"github.com/yaroslavborbat/sandbox-mommy/pkg/logging"
)

var upgradeableMethods = []string{http.MethodGet, http.MethodPost}

func NewAttachREST(serviceAccount types.NamespacedName, sandboxLister corelisters.SandboxLister, client client.GenericClient, restConfig *configrest.Config) *AttachREST {
	return &AttachREST{
		serviceAccount: serviceAccount,
		sandboxLister:  sandboxLister,
		client:         client,
		restConfig:     restConfig,
	}
}

type AttachREST struct {
	serviceAccount types.NamespacedName
	sandboxLister  corelisters.SandboxLister
	client         client.GenericClient
	restConfig     *configrest.Config
}

var (
	_ rest.Storage   = &AttachREST{}
	_ rest.Connecter = &AttachREST{}
)

func (r AttachREST) New() runtime.Object {
	return &subv1alpha1.Attach{}
}

func (r AttachREST) Destroy() {}

func (r AttachREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace := genericreq.NamespaceValue(ctx)
	sandbox, err := r.sandboxLister.Sandboxes(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	if c, _ := condition.GetCondition(sanboxcondition.TypeReady, sandbox.Status.Conditions); c.Status != metav1.ConditionTrue {
		return nil, fmt.Errorf("sandbox %s is not ready", name)
	}

	if err = secrets.load(); err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	nameDepsObj := common.GetFullName(sandbox)
	switch sandbox.Status.Type {
	case v1alpha1.SandboxTypePod:
		pod, err := r.client.Kubernetes().CoreV1().Pods(sandbox.Namespace).Get(ctx, nameDepsObj, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		remoteLocation := r.getPodLocation(pod)
		return r.podHandler(ctx, remoteLocation, responder), nil
	case v1alpha1.SandboxTypeKubevirtVMI:
		kubevirtClient, err := r.client.Kubevirt()
		if err != nil {
			return nil, err
		}
		vmi, err := kubevirtClient.VirtualMachineInstance(sandbox.Namespace).Get(ctx, nameDepsObj, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		remoteLocation, err := r.getKubevirtVMILocation(vmi)
		if err != nil {
			return nil, err
		}

		return r.proxyHandler(remoteLocation, responder)
	case v1alpha1.SandboxTypeDVPVM:
		dvpClient, err := r.client.DVP()
		if err != nil {
			return nil, err
		}
		vm, err := dvpClient.VirtualMachines(sandbox.Namespace).Get(ctx, nameDepsObj, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		remoteLocation, err := r.getDVPVMLocation(vm)
		if err != nil {
			return nil, err
		}
		return r.proxyHandler(remoteLocation, responder)
	default:
		return nil, fmt.Errorf("unknown sandbox type %s", sandbox.Status.Type)
	}
}

func (r AttachREST) podHandler(ctx context.Context, remoteLocation *url.URL, responder rest.Responder) http.Handler {
	var handler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r.setHeaders(request)
		if !isWebSocketRequest(request) {
			responder.Error(apierrors.NewBadRequest("WebSocket upgrade required"))
			return
		}

		conn, err := websocket.Upgrade(writer, request, nil, 0, 0)
		if err != nil {
			responder.Error(apierrors.NewInternalError(fmt.Errorf("failed to upgrade to websocket: %w", err)))
			return
		}
		defer func() {
			if err := conn.Close(); err != nil {
				slog.Error("Failed to close websocket connection", logging.SlogErr(err))
			}
		}()

		if err := r.spdyStream(ctx, conn, remoteLocation); err != nil {
			responder.Error(apierrors.NewInternalError(fmt.Errorf("failed to stream to kube-apiserver %s", err)))
		}
	})
	return handler
}

func (r AttachREST) kubevirtVMIHandler(ctx context.Context, remoteLocation *url.URL, responder rest.Responder) http.Handler {
	var handler http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		r.setHeaders(request)
		if !isWebSocketRequest(request) {
			responder.Error(apierrors.NewBadRequest("WebSocket upgrade required"))
			return
		}

		conn, err := websocket.Upgrade(writer, request, nil, 0, 0)
		if err != nil {
			responder.Error(apierrors.NewInternalError(fmt.Errorf("failed to upgrade to websocket: %w", err)))
			return
		}
		defer conn.Close()

		if err := r.spdyStream(ctx, conn, remoteLocation); err != nil {
			responder.Error(apierrors.NewInternalError(fmt.Errorf("failed to stream to kube-apiserver %s", err)))
		}
	})
	return handler
}

func (r AttachREST) proxyHandler(remoteLocation *url.URL, responder rest.Responder) (http.Handler, error) {
	transport, err := getTransportWithClusterCA(secrets.ca)
	if err != nil {
		return nil, err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.setHeaders(req)
		handler := proxy.NewUpgradeAwareHandler(remoteLocation, transport, false, true, proxy.NewErrorResponder(responder))
		handler.ServeHTTP(w, req)
	}), nil
}

func (r AttachREST) setHeaders(request *http.Request) {
	request.Header.Set("Authorization", "Bearer "+secrets.token)
	request.Header.Set("X-Remote-User", fmt.Sprintf("system:serviceaccount:%s:%s", r.serviceAccount.Namespace, r.serviceAccount.Name))
	request.Header.Set("X-Remote-Group", "system:serviceaccounts")
}

func (r AttachREST) spdyStream(ctx context.Context, wsConn *websocket.Conn, remoteLocation *url.URL) error {
	executor, err := remotecommand.NewSPDYExecutor(r.restConfig, "POST", remoteLocation)
	if err != nil {
		return fmt.Errorf("failed to create SPDY executor: %v", err)
	}

	streamOpts := remotecommand.StreamOptions{
		Stdin:  &wsStreamReader{conn: wsConn},
		Stdout: &wsStreamWriter{conn: wsConn},
		Stderr: &wsStreamWriter{conn: wsConn},
		Tty:    true,
	}

	return executor.StreamWithContext(ctx, streamOpts)
}
func (r AttachREST) NewConnectOptions() (runtime.Object, bool, string) {
	return &subv1alpha1.Attach{}, false, ""
}

func (r AttachREST) ConnectMethods() []string {
	return upgradeableMethods
}

func (r AttachREST) getPodLocation(pod *corev1.Pod) *url.URL {
	const defaultContainerAnnotationName = "kubectl.kubernetes.io/default-container"
	containerName := pod.Spec.Containers[0].Name
	for _, container := range pod.Spec.Containers {
		if pod.Annotations[defaultContainerAnnotationName] == "true" {
			containerName = container.Name
			break
		}
	}
	return r.client.Kubernetes().CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"/bin/bash"},
			Stdin:     true,
			Stdout:    true,
			TTY:       true,
		}, scheme.ParameterCodec).
		URL()
}

func (r AttachREST) getKubevirtVMILocation(pod *virtv1.VirtualMachineInstance) (*url.URL, error) {
	const subresourceURLTpl = "/apis/subresources.kubevirt.io/v1/namespaces/%s/virtualmachineinstances/%s/console"

	kubevirt, err := r.client.Kubevirt()
	if err != nil {
		return nil, err
	}
	return kubevirt.RestClient().
		Post().
		AbsPath(fmt.Sprintf(subresourceURLTpl, pod.Namespace, pod.Name)).
		URL(), nil
}

func (r AttachREST) getDVPVMLocation(vm *dvpcorev1alpha2.VirtualMachine) (*url.URL, error) {
	const vmPathTmpl = "/apis/subresources.virtualization.deckhouse.io/v1alpha2/namespaces/%s/virtualmachines/%s/console"

	restClient, err := restClientForDVPVM(r.restConfig)
	if err != nil {
		return nil, err
	}

	return restClient.
		Post().
		AbsPath(fmt.Sprintf(vmPathTmpl, vm.Namespace, vm.Name)).
		URL(), nil
}

func restClientForDVPVM(c *configrest.Config) (*configrest.RESTClient, error) {
	config := *c
	config.GroupVersion = &dvpsubs.SchemeGroupVersion
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: dvpkubeclient.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	if config.UserAgent == "" {
		config.UserAgent = configrest.DefaultKubernetesUserAgent()
	}

	return configrest.RESTClientFor(&config)
}

func getTransportWithClusterCA(caCert []byte) (*http.Transport, error) {
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.TLSClientConfig = &tls.Config{
		RootCAs: caCertPool,
	}
	return defaultTransport, nil
}
