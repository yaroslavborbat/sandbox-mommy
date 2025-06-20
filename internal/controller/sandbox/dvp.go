package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	dvpcorev1alpha2 "github.com/deckhouse/virtualization/api/core/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	sandboxcondition "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1/sandbox-condition"
	"github.com/yaroslavborbat/sandbox-mommy/internal/common"
	"github.com/yaroslavborbat/sandbox-mommy/internal/featuregate"
)

func NewDVPSandboxer(client client.Client, log *slog.Logger) *DVPSandboxer {
	return &DVPSandboxer{
		client: client,
		log:    log,
	}
}

type DVPSandboxer struct {
	client client.Client
	log    *slog.Logger
}

func (p DVPSandboxer) Create(ctx context.Context, sandbox *v1alpha1.Sandbox, templateSpec *v1alpha1.SandboxTemplateSpec) error {
	if !featuregate.Enabled(featuregate.DVP) {
		return fmt.Errorf("featuregate %s is not enabled", featuregate.DVP)
	}

	vdsForCreate := makeVirtualDisks(sandbox, templateSpec)
	existingVDs, err := p.getVDs(ctx, sandbox)
	if err != nil {
		return err
	}
	existVDMap := make(map[client.ObjectKey]struct{})
	for _, vd := range existingVDs {
		existVDMap[client.ObjectKeyFromObject(vd)] = struct{}{}
	}
	for _, vd := range vdsForCreate {
		if _, exist := existVDMap[client.ObjectKeyFromObject(vd)]; exist {
			continue
		}
		if err = p.client.Create(ctx, vd); err != nil {
			return fmt.Errorf("failed to create pvc %q", client.ObjectKeyFromObject(vd).String())
		}
	}

	vm, err := p.getVM(ctx, sandbox)
	if err != nil {
		return err
	}
	if vm != nil {
		reason := getReasonFromDVPVMPhase(vm.Status.Phase)
		if reason != sandboxcondition.ReasonFailed {
			return nil
		}

		return p.client.Delete(ctx, vm)
	}
	if templateSpec.DVPVMSpec != nil {
		vm = newDVPVM(sandbox, *templateSpec.DVPVMSpec)
		mutateDVPVMVolumes(sandbox, vm, vdsForCreate)

		if err = p.client.Create(ctx, vm); err != nil {
			return fmt.Errorf("failed to create virtual machine %q", client.ObjectKeyFromObject(vm).String())
		}
	}

	return nil
}

func (p DVPSandboxer) Delete(ctx context.Context, sandbox *v1alpha1.Sandbox) error {
	if !featuregate.Enabled(featuregate.DVP) {
		return fmt.Errorf("featuregate %s is not enabled", featuregate.DVP)
	}
	vm, err := p.getVM(ctx, sandbox)
	if err != nil {
		return err
	}
	if vm != nil {
		if err = p.client.Delete(ctx, vm); err != nil {
			return fmt.Errorf("failed to delete virtual machine %q", client.ObjectKeyFromObject(vm).String())
		}
	}
	vds, err := p.getVDs(ctx, sandbox)
	if err != nil {
		return err
	}
	for _, vd := range vds {
		if err = p.client.Delete(ctx, vd); err != nil {
			return fmt.Errorf("failed to delete virtual disk %q", client.ObjectKeyFromObject(vd).String())
		}
	}

	return nil
}

func (p DVPSandboxer) Status(ctx context.Context, sandbox *v1alpha1.Sandbox) (metav1.ConditionStatus, sandboxcondition.Reason, string, error) {
	if !featuregate.Enabled(featuregate.DVP) {
		return "", "", "", fmt.Errorf("featuregate %s is not enabled", featuregate.DVP)
	}

	vm, err := p.getVM(ctx, sandbox)
	if err != nil {
		return metav1.ConditionUnknown, "", "", err
	}
	var (
		status  = metav1.ConditionFalse
		reason  = sandboxcondition.ReasonPending
		message string
	)

	if vm != nil {
		reason = getReasonFromDVPVMPhase(vm.Status.Phase)
		switch reason {
		case sandboxcondition.ReasonReady:
			status = metav1.ConditionTrue
		case sandboxcondition.ReasonFailed:
			var msgs []string
			for _, condition := range vm.Status.Conditions {
				if condition.Message != "" {
					msgs = append(msgs, condition.Message)
				}
			}
			message = strings.Join(msgs, "\n")
		}
	}

	return status, reason, message, nil

}

func (p DVPSandboxer) getVM(ctx context.Context, sandbox *v1alpha1.Sandbox) (*dvpcorev1alpha2.VirtualMachine, error) {
	vm := &dvpcorev1alpha2.VirtualMachine{}
	err := p.client.Get(ctx, client.ObjectKey{Namespace: sandbox.GetNamespace(), Name: common.GetFullName(sandbox)}, vm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get VirtualMachine %w", err)
	}
	return vm, nil
}

func (p DVPSandboxer) getVDs(ctx context.Context, sandbox *v1alpha1.Sandbox) ([]*dvpcorev1alpha2.VirtualDisk, error) {
	vds := &dvpcorev1alpha2.VirtualDiskList{}
	err := p.client.List(ctx, vds, client.InNamespace(sandbox.GetNamespace()))
	if err != nil {
		return nil, fmt.Errorf("failed to list VirtualDisks %w", err)
	}

	var result []*dvpcorev1alpha2.VirtualDisk
	for _, pvc := range vds.Items {
		if metav1.IsControlledBy(&pvc, sandbox) {
			result = append(result, &pvc)
		}
	}

	return result, nil
}

func makeVirtualDisks(sandbox *v1alpha1.Sandbox, templateSpec *v1alpha1.SandboxTemplateSpec) []*dvpcorev1alpha2.VirtualDisk {
	var volumes []*dvpcorev1alpha2.VirtualDisk
	for _, volumeSpec := range templateSpec.Volumes {
		if volumeSpec.VirtualDiskSpec != nil {
			volumes = append(volumes, newVirtualDisk(sandbox, volumeSpec.Name, *volumeSpec.VirtualDiskSpec))
		}
	}
	return volumes
}

func newDVPVM(sandbox *v1alpha1.Sandbox, spec dvpcorev1alpha2.VirtualMachineSpec) *dvpcorev1alpha2.VirtualMachine {
	return &dvpcorev1alpha2.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       dvpcorev1alpha2.VirtualMachineKind,
			APIVersion: dvpcorev1alpha2.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.GetFullName(sandbox),
			Namespace: sandbox.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sandbox, v1alpha1.SchemeGroupVersion.WithKind(v1alpha1.SandboxKind)),
			},
			Labels: map[string]string{
				labelSandboxUID: string(sandbox.GetUID()),
			},
		},
		Spec: spec,
	}
}

func mutateDVPVMVolumes(sandbox *v1alpha1.Sandbox, vm *dvpcorev1alpha2.VirtualMachine, vds []*dvpcorev1alpha2.VirtualDisk) {
	vdsMap := make(map[string]struct{})
	for _, vd := range vds {
		vdsMap[vd.Name] = struct{}{}
	}

	for i, ref := range vm.Spec.BlockDeviceRefs {
		if ref.Kind == dvpcorev1alpha2.VirtualDiskKind {
			fullName := getFullVDName(ref.Name, sandbox)
			if _, ok := vdsMap[fullName]; ok {
				vm.Spec.BlockDeviceRefs[i].Name = fullName
			}
		}
	}
}

func newVirtualDisk(sandbox *v1alpha1.Sandbox, name string, spec dvpcorev1alpha2.VirtualDiskSpec) *dvpcorev1alpha2.VirtualDisk {
	return &dvpcorev1alpha2.VirtualDisk{
		TypeMeta: metav1.TypeMeta{
			Kind:       dvpcorev1alpha2.VirtualDiskKind,
			APIVersion: dvpcorev1alpha2.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFullVDName(name, sandbox),
			Namespace: sandbox.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(sandbox, v1alpha1.SchemeGroupVersion.WithKind(v1alpha1.SandboxKind)),
			},
			Labels: map[string]string{
				labelSandboxUID: string(sandbox.GetUID()),
			},
		},
		Spec: spec,
	}
}

const vdNamePrefix = "sandbox-vd-"

func getFullVDName(name string, sandbox *v1alpha1.Sandbox) string {
	return fmt.Sprintf("%s%s-%s", vdNamePrefix, sandbox.GetUID(), name)
}

func getReasonFromDVPVMPhase(phase dvpcorev1alpha2.MachinePhase) sandboxcondition.Reason {
	reason, ok := mapDVPVMToSandboxReadyReason[phase]
	if !ok {
		return sandboxcondition.ReasonPending
	}
	return reason
}

var mapDVPVMToSandboxReadyReason = map[dvpcorev1alpha2.MachinePhase]sandboxcondition.Reason{
	dvpcorev1alpha2.MachinePending:     sandboxcondition.ReasonPending,
	dvpcorev1alpha2.MachineRunning:     sandboxcondition.ReasonReady,
	dvpcorev1alpha2.MachineTerminating: sandboxcondition.ReasonFailed,
	dvpcorev1alpha2.MachineStopped:     sandboxcondition.ReasonFailed,
	dvpcorev1alpha2.MachineStopping:    sandboxcondition.ReasonFailed,
	dvpcorev1alpha2.MachineStarting:    sandboxcondition.ReasonPending,
	dvpcorev1alpha2.MachineMigrating:   sandboxcondition.ReasonPending,
	dvpcorev1alpha2.MachinePause:       sandboxcondition.ReasonFailed,
	dvpcorev1alpha2.MachineDegraded:    sandboxcondition.ReasonFailed,
}
