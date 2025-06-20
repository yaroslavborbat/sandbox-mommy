package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	virtv1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	sandboxcondition "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1/sandbox-condition"
	"github.com/yaroslavborbat/sandbox-mommy/internal/common"
	"github.com/yaroslavborbat/sandbox-mommy/internal/featuregate"
)

func NewKubevirtSandboxer(client client.Client, log *slog.Logger) *KubevirtSandboxer {
	return &KubevirtSandboxer{
		client: client,
		pvcManager: pvcManager{
			client: client,
		},
		log: log,
	}
}

type KubevirtSandboxer struct {
	client     client.Client
	pvcManager pvcManager
	log        *slog.Logger
}

func (p KubevirtSandboxer) Create(ctx context.Context, sandbox *v1alpha1.Sandbox, templateSpec *v1alpha1.SandboxTemplateSpec) error {
	if !featuregate.Enabled(featuregate.Kubevirt) {
		return fmt.Errorf("featuregate %s is not enabled", featuregate.Kubevirt)
	}

	dvsForCreate := makeDataVolumes(sandbox, templateSpec)
	existingVDs, err := p.getDVs(ctx, sandbox)
	if err != nil {
		return err
	}
	existDVMap := make(map[client.ObjectKey]struct{})
	for _, pvc := range existingVDs {
		existDVMap[client.ObjectKeyFromObject(pvc)] = struct{}{}
	}
	for _, pvc := range dvsForCreate {
		if _, exist := existDVMap[client.ObjectKeyFromObject(pvc)]; exist {
			continue
		}
		if err = p.client.Create(ctx, pvc); err != nil {
			return fmt.Errorf("failed to create data volume %q", client.ObjectKeyFromObject(pvc).String())
		}
	}

	pvcsForCreate := makePersistentVolumeClaims(sandbox, templateSpec)
	if err := p.pvcManager.createPVCs(ctx, sandbox, pvcsForCreate); err != nil {
		return err
	}

	vmi, err := p.getVMI(ctx, sandbox)
	if err != nil {
		return err
	}
	if vmi != nil {
		reason := getReasonFromKubevirtVMIPhase(vmi.Status.Phase)
		if reason != sandboxcondition.ReasonFailed {
			return nil
		}

		return p.client.Delete(ctx, vmi)
	}
	if templateSpec.KubevirtVMISpec != nil {
		vmi = newKubevirtVMI(sandbox, *templateSpec.KubevirtVMISpec)
		mutateKubevirtVMIVolumes(sandbox, vmi, dvsForCreate, pvcsForCreate)
		if err = p.client.Create(ctx, vmi); err != nil {
			return fmt.Errorf("failed to create virtual machine instance %q", client.ObjectKeyFromObject(vmi).String())
		}
	}

	return nil
}

func (p KubevirtSandboxer) Delete(ctx context.Context, sandbox *v1alpha1.Sandbox) error {
	if !featuregate.Enabled(featuregate.Kubevirt) {
		return fmt.Errorf("featuregate %s is not enabled", featuregate.Kubevirt)
	}
	vmi, err := p.getVMI(ctx, sandbox)
	if err != nil {
		return err
	}
	if vmi != nil {
		if err = p.client.Delete(ctx, vmi); err != nil {
			return fmt.Errorf("failed to delete virtual machine instance %q", client.ObjectKeyFromObject(vmi).String())
		}
	}
	dvs, err := p.getDVs(ctx, sandbox)
	if err != nil {
		return err
	}
	for _, vd := range dvs {
		if err = p.client.Delete(ctx, vd); err != nil {
			return fmt.Errorf("failed to delete data volume %q", client.ObjectKeyFromObject(vd).String())
		}
	}

	return p.pvcManager.deletePVCs(ctx, sandbox)
}

func (p KubevirtSandboxer) Status(ctx context.Context, sandbox *v1alpha1.Sandbox) (metav1.ConditionStatus, sandboxcondition.Reason, string, error) {
	if !featuregate.Enabled(featuregate.Kubevirt) {
		return "", "", "", fmt.Errorf("featuregate %s is not enabled", featuregate.Kubevirt)
	}

	vmi, err := p.getVMI(ctx, sandbox)
	if err != nil {
		return metav1.ConditionUnknown, "", "", err
	}
	var (
		status  = metav1.ConditionFalse
		reason  = sandboxcondition.ReasonPending
		message string
	)

	if vmi != nil {
		reason = getReasonFromKubevirtVMIPhase(vmi.Status.Phase)
		switch reason {
		case sandboxcondition.ReasonReady:
			status = metav1.ConditionTrue
		case sandboxcondition.ReasonFailed:
			var msgs []string
			for _, condition := range vmi.Status.Conditions {
				if condition.Message != "" {
					msgs = append(msgs, condition.Message)
				}
			}
			message = strings.Join(msgs, "\n")
		}
	}

	return status, reason, message, nil

}

func (p KubevirtSandboxer) getVMI(ctx context.Context, sandbox *v1alpha1.Sandbox) (*virtv1.VirtualMachineInstance, error) {
	vmi := &virtv1.VirtualMachineInstance{}
	err := p.client.Get(ctx, client.ObjectKey{Namespace: sandbox.GetNamespace(), Name: common.GetFullName(sandbox)}, vmi)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get VirtualMachineInstance %w", err)
	}
	return vmi, nil
}

func (p KubevirtSandboxer) getDVs(ctx context.Context, sandbox *v1alpha1.Sandbox) ([]*cdiv1beta1.DataVolume, error) {
	dvs := &cdiv1beta1.DataVolumeList{}
	err := p.client.List(ctx, dvs, client.InNamespace(sandbox.GetNamespace()))
	if err != nil {
		return nil, fmt.Errorf("failed to list VirtualDisks %w", err)
	}

	var result []*cdiv1beta1.DataVolume
	for _, dv := range dvs.Items {
		if metav1.IsControlledBy(&dv, sandbox) {
			result = append(result, &dv)
		}
	}

	return result, nil
}

func (p KubevirtSandboxer) getPVCs(ctx context.Context, sandbox *v1alpha1.Sandbox) ([]*corev1.PersistentVolumeClaim, error) {
	pvcs := &corev1.PersistentVolumeClaimList{}
	err := p.client.List(ctx, pvcs, client.InNamespace(sandbox.GetNamespace()))
	if err != nil {
		return nil, fmt.Errorf("failed to list PVCs %w", err)
	}

	var result []*corev1.PersistentVolumeClaim
	for _, pvc := range pvcs.Items {
		if metav1.IsControlledBy(&pvc, sandbox) {
			result = append(result, &pvc)
		}
	}

	return result, nil
}

func makeDataVolumes(sandbox *v1alpha1.Sandbox, templateSpec *v1alpha1.SandboxTemplateSpec) []*cdiv1beta1.DataVolume {
	var volumes []*cdiv1beta1.DataVolume
	for _, volumeSpec := range templateSpec.Volumes {
		if volumeSpec.DataVolumeSpec != nil {
			volumes = append(volumes, newDataVolume(sandbox, volumeSpec.Name, *volumeSpec.DataVolumeSpec))
		}
	}
	return volumes
}

func newKubevirtVMI(sandbox *v1alpha1.Sandbox, spec virtv1.VirtualMachineInstanceSpec) *virtv1.VirtualMachineInstance {
	return &virtv1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineInstance",
			APIVersion: virtv1.SchemeGroupVersion.String(),
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

func newDataVolume(sandbox *v1alpha1.Sandbox, name string, spec cdiv1beta1.DataVolumeSpec) *cdiv1beta1.DataVolume {
	return &cdiv1beta1.DataVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DataVolume",
			APIVersion: cdiv1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFullDVName(name, sandbox),
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

func mutateKubevirtVMIVolumes(sandbox *v1alpha1.Sandbox, vmi *virtv1.VirtualMachineInstance, dvs []*cdiv1beta1.DataVolume, pvcs []*corev1.PersistentVolumeClaim) {
	dvsMap := make(map[string]struct{})
	for _, dv := range dvs {
		dvsMap[dv.Name] = struct{}{}
	}
	pvcsMap := make(map[string]struct{})
	for _, pvc := range pvcs {
		pvcsMap[pvc.Name] = struct{}{}
	}

	for i, volume := range vmi.Spec.Volumes {
		if volume.DataVolume != nil {
			fullName := getFullDVName(volume.DataVolume.Name, sandbox)
			if _, ok := dvsMap[fullName]; !ok {
				continue
			}
			vmi.Spec.Volumes[i].DataVolume.Name = fullName
		}
		if volume.PersistentVolumeClaim != nil {
			fullName := getFullPVCName(volume.PersistentVolumeClaim.ClaimName, sandbox)
			if _, ok := pvcsMap[fullName]; !ok {
				continue
			}
		}
	}

	for i, disk := range vmi.Spec.Domain.Devices.Disks {
		if disk.Disk != nil {
			fullDVName := getFullDVName(disk.Name, sandbox)
			if _, ok := dvsMap[fullDVName]; ok {
				vmi.Spec.Domain.Devices.Disks[i].Name = fullDVName
			}
			fullPVCName := getFullPVCName(disk.Name, sandbox)
			if _, ok := pvcsMap[fullPVCName]; ok {
				vmi.Spec.Domain.Devices.Disks[i].Name = fullPVCName
			}
		}
	}

}

const dvNamePrefix = "sandbox-dv-"

func getFullDVName(name string, sandbox *v1alpha1.Sandbox) string {
	return fmt.Sprintf("%s%s-%s", dvNamePrefix, sandbox.GetUID(), name)
}

func getReasonFromKubevirtVMIPhase(phase virtv1.VirtualMachineInstancePhase) sandboxcondition.Reason {
	reason, ok := mapKubevirtVMIToSandboxReadyReason[phase]
	if !ok {
		return sandboxcondition.ReasonPending
	}
	return reason
}

var mapKubevirtVMIToSandboxReadyReason = map[virtv1.VirtualMachineInstancePhase]sandboxcondition.Reason{
	virtv1.VmPhaseUnset: sandboxcondition.ReasonPending,
	virtv1.Pending:      sandboxcondition.ReasonPending,
	virtv1.Scheduling:   sandboxcondition.ReasonPending,
	virtv1.Scheduled:    sandboxcondition.ReasonPending,
	virtv1.Running:      sandboxcondition.ReasonReady,
	virtv1.Succeeded:    sandboxcondition.ReasonFailed,
	virtv1.Failed:       sandboxcondition.ReasonFailed,
	virtv1.Unknown:      sandboxcondition.ReasonFailed,
}
