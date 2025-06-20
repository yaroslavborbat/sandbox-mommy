package sandbox

import (
	"context"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1"
	sandboxcondition "github.com/yaroslavborbat/sandbox-mommy/api/core/v1alpha1/sandbox-condition"
	"github.com/yaroslavborbat/sandbox-mommy/internal/common"
)

func NewPodSandboxer(client client.Client, log *slog.Logger) *PodSandboxer {
	return &PodSandboxer{
		client: client,
		pvcManager: pvcManager{
			client: client,
		},
		log: log,
	}
}

type PodSandboxer struct {
	client     client.Client
	pvcManager pvcManager
	log        *slog.Logger
}

func (p PodSandboxer) Create(ctx context.Context, sandbox *v1alpha1.Sandbox, templateSpec *v1alpha1.SandboxTemplateSpec) error {
	pvcsForCreate := makePersistentVolumeClaims(sandbox, templateSpec)
	if err := p.pvcManager.createPVCs(ctx, sandbox, pvcsForCreate); err != nil {
		return err
	}

	pod, err := p.getPOD(ctx, sandbox)
	if err != nil {
		return err
	}
	if pod != nil {
		reason := getReasonFromPodPhase(pod.Status.Phase)
		if reason != sandboxcondition.ReasonFailed {
			return nil
		}

		return p.client.Delete(ctx, pod)
	}
	if templateSpec.PodSpec != nil {
		pod = newPod(sandbox, *templateSpec.PodSpec)
		mutatePodPVCs(sandbox, pod, pvcsForCreate)
		if err = p.client.Create(ctx, pod); err != nil {
			return fmt.Errorf("failed to create pod %q", client.ObjectKeyFromObject(pod).String())
		}
	}

	return nil
}

func (p PodSandboxer) Delete(ctx context.Context, sandbox *v1alpha1.Sandbox) error {
	pod, err := p.getPOD(ctx, sandbox)
	if err != nil {
		return err
	}
	if pod != nil {
		if err = p.client.Delete(ctx, pod); err != nil {
			return fmt.Errorf("failed to delete pod %q", client.ObjectKeyFromObject(pod).String())
		}
	}
	return p.pvcManager.deletePVCs(ctx, sandbox)
}

func (p PodSandboxer) Status(ctx context.Context, sandbox *v1alpha1.Sandbox) (metav1.ConditionStatus, sandboxcondition.Reason, string, error) {
	pod, err := p.getPOD(ctx, sandbox)
	if err != nil {
		return metav1.ConditionUnknown, "", "", err
	}
	var (
		status  = metav1.ConditionFalse
		reason  = sandboxcondition.ReasonPending
		message string
	)

	if pod != nil {
		reason = getReasonFromPodPhase(pod.Status.Phase)
		switch reason {
		case sandboxcondition.ReasonReady:
			status = metav1.ConditionTrue
		case sandboxcondition.ReasonFailed:
			message = pod.Status.Message
		}
	}

	return status, reason, message, nil

}

func (p PodSandboxer) getPOD(ctx context.Context, sandbox *v1alpha1.Sandbox) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := p.client.Get(ctx, client.ObjectKey{Namespace: sandbox.GetNamespace(), Name: common.GetFullName(sandbox)}, pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get pod %w", err)
	}
	return pod, nil
}

func getReasonFromPodPhase(phase corev1.PodPhase) sandboxcondition.Reason {
	reason, ok := mapPodToSandboxReadyReason[phase]
	if !ok {
		return sandboxcondition.ReasonPending
	}
	return reason
}

var mapPodToSandboxReadyReason = map[corev1.PodPhase]sandboxcondition.Reason{
	corev1.PodPending:   sandboxcondition.ReasonPending,
	corev1.PodRunning:   sandboxcondition.ReasonReady,
	corev1.PodFailed:    sandboxcondition.ReasonFailed,
	corev1.PodSucceeded: sandboxcondition.ReasonFailed,
	corev1.PodUnknown:   sandboxcondition.ReasonFailed,
}

func newPod(sandbox *v1alpha1.Sandbox, spec corev1.PodSpec) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
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

func mutatePodPVCs(sandbox *v1alpha1.Sandbox, pod *corev1.Pod, pvcs []*corev1.PersistentVolumeClaim) {
	pvcsMap := make(map[string]struct{})
	for _, pvc := range pvcs {
		pvcsMap[pvc.Name] = struct{}{}
	}

	for i, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			fullName := getFullPVCName(volume.PersistentVolumeClaim.ClaimName, sandbox)
			if _, ok := pvcsMap[fullName]; ok {
				pod.Spec.Volumes[i].PersistentVolumeClaim.ClaimName = fullName
			}
		}
	}
	for i, ctr := range pod.Spec.Containers {
		for j, mount := range ctr.VolumeMounts {
			fullName := getFullPVCName(mount.Name, sandbox)
			if _, ok := pvcsMap[fullName]; ok {
				pod.Spec.Containers[i].VolumeMounts[j].Name = fullName
			}
		}
	}
}
