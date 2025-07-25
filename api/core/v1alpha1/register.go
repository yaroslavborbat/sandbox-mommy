package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/yaroslavborbat/sandbox-mommy/api/core"
)

const (
	Version = "v1alpha1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: core.GroupName, Version: Version}

	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	AddToScheme = SchemeBuilder.AddToScheme
)

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func GroupVersionResource(resource string) schema.GroupVersionResource {
	return SchemeGroupVersion.WithResource(resource)
}

// Adds the list of known types to the Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Sandbox{},
		&SandboxList{},
		&SandboxTemplate{},
		&SandboxTemplateList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
