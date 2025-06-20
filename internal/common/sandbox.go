package common

import "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"

func DetectSandboxType(spec *v1alpha1.SandboxTemplateSpec) v1alpha1.SandboxType {
	if spec == nil {
		return ""
	}
	switch {
	case spec.PodSpec != nil:
		return v1alpha1.SandboxTypePod
	case spec.KubevirtVMISpec != nil:
		return v1alpha1.SandboxTypeKubevirtVMI
	case spec.DVPVMSpec != nil:
		return v1alpha1.SandboxTypeDVPVM
	default:
		return ""
	}
}
