package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func rollbackWorkloadARMAffinity(selectedWorkloads []Workload) {
	for _, workload := range selectedWorkloads {
		var err error
		switch workload.Kind {
		case WorkloadDeployment:
			err = rollbackDeploymentARMAffinity(&workload)
		case WorkloadStatefulSet:
			err = rollbackStatefulSetARMAffinity(&workload)
		}
		if err != nil {
			fmt.Printf("Failed to rollback %s workload %s/%s, err: %v\n", workload.Kind,
				workload.Namespace, workload.Name, err)
		}
	}
}

func rollbackDeploymentARMAffinity(workload *Workload) error {
	ctx := context.Background()
	deployment, err := kubeClient.AppsV1().Deployments(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get deployment: %w", err)
	}

	newDeployment := deployment.DeepCopy()
	newDeployment.Spec.Template.Spec.Affinity = ensurePreferAffinity(newDeployment.Spec.Template.Spec.Affinity)

	newDeployment.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
		RemoveArm64Preference(newDeployment.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	newDeployment.Spec.Template.Spec.Tolerations = RemoveARM64Toleration(newDeployment.Spec.Template.Spec.Tolerations)

	return patchResource(ctx, deployment, newDeployment, workload.Namespace, workload.Name, workload.Kind)
}

func rollbackStatefulSetARMAffinity(workload *Workload) error {
	ctx := context.Background()

	ss, err := kubeClient.AppsV1().
		StatefulSets(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get statefulset: %w", err)
	}

	newSS := ss.DeepCopy()
	newSS.Spec.Template.Spec.Affinity = ensurePreferAffinity(newSS.Spec.Template.Spec.Affinity)

	newSS.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
		RemoveArm64Preference(newSS.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	newSS.Spec.Template.Spec.Tolerations = RemoveARM64Toleration(newSS.Spec.Template.Spec.Tolerations)

	return patchResource(ctx, ss, newSS, workload.Namespace, workload.Name, workload.Kind)
}
