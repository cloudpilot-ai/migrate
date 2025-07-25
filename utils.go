package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func patchResource(ctx context.Context, originalObj, updatedObj interface{}, namespace, name string, kind WorkloadKind) error {
	originalBytes, err := json.Marshal(originalObj)
	if err != nil {
		return fmt.Errorf("marshal original: %w", err)
	}
	updatedBytes, err := json.Marshal(updatedObj)
	if err != nil {
		return fmt.Errorf("marshal updated: %w", err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(originalBytes, updatedBytes)
	if err != nil {
		return fmt.Errorf("create merge patch: %w", err)
	}

	switch kind {
	case WorkloadDeployment:
		err = backoff.Retry(func() error {
			_, patchErr := kubeClient.AppsV1().Deployments(namespace).
				Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
			return patchErr
		}, DefaultBackoff(ctx))
	case WorkloadStatefulSet:
		err = backoff.Retry(func() error {
			_, patchErr := kubeClient.AppsV1().StatefulSets(namespace).
				Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
			return patchErr
		}, DefaultBackoff(ctx))
	default:
		return fmt.Errorf("unsupported resource kind: %s", kind)
	}

	if err != nil {
		return fmt.Errorf("failed to patch %s: %w", kind, err)
	}

	fmt.Printf("Patched workload %s %s/%s successfully\n", kind, namespace, name)
	return nil
}

func DefaultBackoff(ctx context.Context) backoff.BackOffContext {
	return backoff.WithContext(backoff.WithMaxRetries(backoff.NewConstantBackOff(1*time.Second), 5), ctx)
}

func ensurePreferAffinity(sourceAff *corev1.Affinity) *corev1.Affinity {
	aff := sourceAff.DeepCopy()
	if aff == nil {
		aff = &corev1.Affinity{}
	}
	if aff.NodeAffinity == nil {
		aff.NodeAffinity = &corev1.NodeAffinity{}
	}
	if aff.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil {
		aff.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []corev1.PreferredSchedulingTerm{}
	}
	return aff
}
