package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const SandboxKind = "Sandbox"

// The Sandbox resource describes configuration sandbox.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories={sandbox-mommy},scope=Namespaced,shortName={sb,sbs},singular=sandbox
// +kubebuilder:printcolumn:name="TTL",type="string",JSONPath=".spec.ttl",description="Sandbox TTL."
// +kubebuilder:printcolumn:name="Template",type="string",JSONPath=".spec.template",description="Sandbox template name."
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".status.type",description="Sandbox type."
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason",description="Sandbox status."
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time of resource creation."
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Sandbox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SandboxSpec   `json:"spec,omitempty"`
	Status SandboxStatus `json:"status,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="size(self.template) != 0 || has(self.templateSpec)",message="Either template or templateSpec must be specified"
// +kubebuilder:validation:XValidation:rule="!(size(self.template) != 0 && has(self.templateSpec))",message="Only one of template or templateSpec must be specified"
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message=".spec is immutable"
type SandboxSpec struct {
	// Name of the sandbox template to use.
	Template string `json:"template,omitempty"`
	// TemplateSpec is the spec of the sandbox template.
	TemplateSpec *SandboxTemplateSpec `json:"templateSpec,omitempty"`
	// TTL is the time after which the sandbox will be automatically deleted
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Format=duration
	// +kubebuilder:default:="60m"
	TTL metav1.Duration `json:"ttl"`
}

type SandboxStatus struct {
	Type       SandboxType        `json:"type,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:validation:Enum:={"", Pod,DVP/VirtualMachine,Kubevirt/VirtualMachineInstance}
type SandboxType string

const (
	SandboxTypePod         SandboxType = "Pod"
	SandboxTypeDVPVM       SandboxType = "DVP/VirtualMachine"
	SandboxTypeKubevirtVMI SandboxType = "Kubevirt/VirtualMachineInstance"
)

// The SandboxList resource describes a list of Sandbox resources.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SandboxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Sandbox `json:"items"`
}
