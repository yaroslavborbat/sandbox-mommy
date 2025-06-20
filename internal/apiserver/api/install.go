package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	configrest "k8s.io/client-go/rest"

	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/registry/sandbox/client"

	corelisters "github.com/yaroslavborbat/sandbox-mommy/api/client/generated/listers/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/api/subresources"
	"github.com/yaroslavborbat/sandbox-mommy/api/subresources/install"
	subv1alpha1 "github.com/yaroslavborbat/sandbox-mommy/api/subresources/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/apiserver/registry/sandbox/storage"
)

var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)
)

func init() {
	install.Install(Scheme)
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

func Build(storage *storage.Storage) genericapiserver.APIGroupInfo {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(subresources.GroupName, Scheme, ParameterCodec, Codecs)
	resources := map[string]rest.Storage{
		"sandboxes":        storage,
		"sandboxes/attach": storage.AttachREST(),
	}
	apiGroupInfo.VersionedResourcesStorageMap[subv1alpha1.SchemeGroupVersion.Version] = resources
	return apiGroupInfo
}

func Install(
	sandboxLister corelisters.SandboxLister,
	server *genericapiserver.GenericAPIServer,
	client client.GenericClient,
	serviceAccount types.NamespacedName,
	restConfig *configrest.Config,
) error {
	sandboxStorage := storage.NewStorage(serviceAccount, sandboxLister, client, restConfig)
	info := Build(sandboxStorage)
	return server.InstallAPIGroup(&info)
}
