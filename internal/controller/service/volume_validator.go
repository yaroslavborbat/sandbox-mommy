package service

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	"github.com/yaroslavborbat/sandbox-mommy/internal/featuregate"
)

type VolumesValidator struct{}

func (v *VolumesValidator) Validate(_ context.Context, templateSpec *v1alpha1.SandboxTemplateSpec) (admission.Warnings, error) {

	volumeMap := make(map[string]struct{})

	for _, volume := range templateSpec.Volumes {
		if _, exist := volumeMap[volume.Name]; exist {
			return admission.Warnings{}, fmt.Errorf("volume %s already exists", volume.Name)
		}
		volumeMap[volume.Name] = struct{}{}
	}

	return admission.Warnings{}, nil
}

type TypeValidator struct{}

func (v *TypeValidator) Validate(_ context.Context, templateSpec *v1alpha1.SandboxTemplateSpec) (admission.Warnings, error) {
	switch {
	case templateSpec.KubevirtVMISpec != nil:
		if !featuregate.Enabled(featuregate.Kubevirt) {
			return admission.Warnings{}, fmt.Errorf("featuregate %s is not enabled", featuregate.Kubevirt)
		}
	case templateSpec.DVPVMSpec != nil:
		if !featuregate.Enabled(featuregate.DVP) {
			return admission.Warnings{}, fmt.Errorf("featuregate %s is not enabled", featuregate.DVP)
		}
	}

	for _, volume := range templateSpec.Volumes {
		switch {
		case volume.DataVolumeSpec != nil:
			if !featuregate.Enabled(featuregate.Kubevirt) {
				return admission.Warnings{}, fmt.Errorf("featuregate %s is not enabled", featuregate.Kubevirt)
			}
		case volume.VirtualDiskSpec != nil:
			if !featuregate.Enabled(featuregate.DVP) {
				return admission.Warnings{}, fmt.Errorf("featuregate %s is not enabled", featuregate.DVP)
			}
		}
	}

	return admission.Warnings{}, nil
}
