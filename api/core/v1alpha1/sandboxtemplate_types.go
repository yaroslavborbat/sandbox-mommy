package v1alpha1

import (
	dvpcorev1alpha2 "github.com/deckhouse/virtualization/api/core/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

const SandboxTemplateKind = "SandboxTemplate"

// The SandboxTemplate resource describes configuration sandbox template.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories={sandbox-mommy},scope=Cluster,shortName={sbt,sbts},singular=sandboxtemplate
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".status.type",description="SandboxTemplate type."
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason",description="SandboxTemplate status."
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time of resource creation."
// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SandboxTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SandboxTemplateSpec   `json:"spec,omitempty"`
	Status SandboxTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.podSpec) || has(self.kubevirtVMISpec) || has(self.dvpVMSpec)",message="Either podSpec,kubevirtVMISpec or dvpVMSpec must be specified"
// +kubebuilder:validation:XValidation:rule="!(has(self.podSpec) && has(self.kubevirtVMISpec) && has(self.dvpVMSpec))",message="Only one of podSpec,kubevirtVMISpec or dvpVMSpecmust be specified"
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message=".spec is immutable"
type SandboxTemplateSpec struct {
	// PodSpec is the spec of the pod to run in the sandbox.
	PodSpec *corev1.PodSpec `json:"podSpec,omitempty"`
	// KubevirtVMISpec is the spec of the kubevirt virtual machine instance to run in the sandbox.
	KubevirtVMISpec *virtv1.VirtualMachineInstanceSpec `json:"kubevirtVMISpec,omitempty"`
	// DVPVMSpec is the spec of the dvp virtual machine to run in the sandbox.
	DVPVMSpec *dvpcorev1alpha2.VirtualMachineSpec `json:"dvpVMSpec,omitempty"`
	// Volumes is the list of volumes to create and mount in the sandbox.
	Volumes []SandboxVolumeSpec `json:"volumes,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.pvcSpec) || has(self.dataVolumeSpec) || has(self.virtualDiskSpec)",message="Either pvcSpec, dataVolumeSpec or virtualDiskSpec must be specified"
// +kubebuilder:validation:XValidation:rule="!(has(self.pvcSpec) && has(self.dataVolumeSpec) && has(self.virtualDiskSpec))",message="Only one of pvcSpec, dataVolumeSpec or virtualDiskSpec must be specified"
type SandboxVolumeSpec struct {
	// Name is the name of the volume.
	Name string `json:"name"`
	// PVCSpec is the spec of the persistent volume claim to use.
	PVCSpec *corev1.PersistentVolumeClaimSpec `json:"pvcSpec,omitempty"`
	// DataVolumeSpec is the spec of the data volume to use.
	DataVolumeSpec *cdiv1beta1.DataVolumeSpec `json:"dataVolumeSpec,omitempty"`
	// VirtualDiskSpec is the spec of the virtual disk to use.
	VirtualDiskSpec *dvpcorev1alpha2.VirtualDiskSpec `json:"virtualDiskSpec,omitempty"`
}
type SandboxTemplateStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Type       SandboxType        `json:"type,omitempty"`
}

// The SandboxTemplateList resource describes a list of SandboxTemplate resources.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SandboxTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SandboxTemplate `json:"items"`
}
