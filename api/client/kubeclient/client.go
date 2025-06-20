package kubeclient

import (
	"io"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"github.com/yaroslavborbat/sandbox-mommy/api/client/generated/clientset/versioned"
	sandboxv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/client/generated/clientset/versioned/typed/core/v1alpha1"
	coreinstall "github.com/yaroslavborbat/sandbox-mommy/api/core/install"
	subinstall "github.com/yaroslavborbat/sandbox-mommy/api/subresources/install"
	subv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/subresources/v1alpha1"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	coreinstall.Install(Scheme)
	subinstall.Install(Scheme)
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

type Client interface {
	RESTConfig() *rest.Config
	RESTClient() *rest.RESTClient
	Sandboxes(namespace string) SandboxInterface
	SandboxTemplates() sandboxv1alpha1.SandboxTemplateInterface
}
type SandboxInterface interface {
	sandboxv1alpha1.SandboxInterface
	Attach(name string, options *subv1alpha1.Attach) (StreamInterface, error)
}

type StreamInterface interface {
	Stream(options StreamOptions) error
	AsConn() net.Conn
}
type StreamOptions struct {
	In  io.Reader
	Out io.Writer
}

type client struct {
	config        *rest.Config
	shallowCopy   *rest.Config
	restClient    *rest.RESTClient
	sandboxClient *versioned.Clientset
}

func (c client) RESTConfig() *rest.Config {
	return rest.CopyConfig(c.config)
}

func (c client) RESTClient() *rest.RESTClient {
	return c.restClient
}

func (c client) Sandboxes(namespace string) SandboxInterface {
	return &sandbox{
		config:           c.config,
		resource:         "sandboxes",
		namespace:        namespace,
		restClient:       c.restClient,
		SandboxInterface: c.sandboxClient.SandboxV1alpha1().Sandboxes(namespace),
	}
}

func (c client) SandboxTemplates() sandboxv1alpha1.SandboxTemplateInterface {
	return c.sandboxClient.SandboxV1alpha1().SandboxTemplates()
}
