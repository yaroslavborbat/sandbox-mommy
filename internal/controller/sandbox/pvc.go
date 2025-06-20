package sandbox

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
)

func makePersistentVolumeClaims(sandbox *v1alpha1.Sandbox, templateSpec *v1alpha1.SandboxTemplateSpec) []*corev1.PersistentVolumeClaim {
	var volumes []*corev1.PersistentVolumeClaim
	for _, volumeSpec := range templateSpec.Volumes {
		if volumeSpec.PVCSpec != nil {
			volumes = append(volumes, newPVC(sandbox, volumeSpec.Name, *volumeSpec.PVCSpec))
		}
	}
	return volumes
}

func newPVC(sandbox *v1alpha1.Sandbox, name string, spec corev1.PersistentVolumeClaimSpec) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getFullPVCName(name, sandbox),
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

const pvcNamePrefix = "sandbox-pvc-"

func getFullPVCName(name string, sandbox *v1alpha1.Sandbox) string {
	return fmt.Sprintf("%s%s-%s", pvcNamePrefix, sandbox.GetUID(), name)
}

func getPVCs(ctx context.Context, sandbox *v1alpha1.Sandbox, c client.Client) ([]*corev1.PersistentVolumeClaim, error) {
	pvcs := &corev1.PersistentVolumeClaimList{}
	err := c.List(ctx, pvcs, client.InNamespace(sandbox.GetNamespace()))
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

type pvcManager struct {
	client client.Client
}

func (m pvcManager) createPVCs(ctx context.Context, sandbox *v1alpha1.Sandbox, pvcsForCreate []*corev1.PersistentVolumeClaim) error {
	existingPVCs, err := m.getPVCs(ctx, sandbox)
	if err != nil {
		return err
	}
	existPVCMap := make(map[client.ObjectKey]struct{})
	for _, pvc := range existingPVCs {
		existPVCMap[client.ObjectKeyFromObject(pvc)] = struct{}{}
	}
	for _, pvc := range pvcsForCreate {
		if _, exist := existPVCMap[client.ObjectKeyFromObject(pvc)]; exist {
			continue
		}
		if err = m.client.Create(ctx, pvc); err != nil {
			return fmt.Errorf("failed to create pvc %q", client.ObjectKeyFromObject(pvc).String())
		}
	}

	return nil
}

func (m pvcManager) deletePVCs(ctx context.Context, sandbox *v1alpha1.Sandbox) error {
	pvcs, err := m.getPVCs(ctx, sandbox)
	if err != nil {
		return err
	}
	for _, pvc := range pvcs {
		if err = m.client.Delete(ctx, pvc); err != nil {
			return fmt.Errorf("failed to delete pvc %q", client.ObjectKeyFromObject(pvc).String())
		}
	}
	return nil
}

func (m pvcManager) getPVCs(ctx context.Context, sandbox *v1alpha1.Sandbox) ([]*corev1.PersistentVolumeClaim, error) {
	pvcs := &corev1.PersistentVolumeClaimList{}
	err := m.client.List(ctx, pvcs, client.InNamespace(sandbox.GetNamespace()))
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
