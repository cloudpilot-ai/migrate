package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func patchWorkloadARMAffinity(selectedWorkloads []Workload) {
	for _, workload := range selectedWorkloads {
		var err error
		switch workload.Kind {
		case WorkloadDeployment:
			err = patchDeploymentARMAffinity(&workload)
		case WorkloadStatefulSet:
			err = patchStatefulSetARMAffinity(&workload)
		}
		if err != nil {
			fmt.Printf("Failed to patch %s workload %s/%s: %v\n", workload.Kind,
				workload.Namespace, workload.Name, err)
		}
	}
}

func patchDeploymentARMAffinity(workload *Workload) error {
	ctx := context.Background()
	deployment, err := kubeClient.AppsV1().Deployments(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get deployment: %w", err)
	}

	newDeployment := deployment.DeepCopy()
	newDeployment.Spec.Template.Spec.Affinity = ensurePreferAffinity(newDeployment.Spec.Template.Spec.Affinity)

	if HasArm64Preference(newDeployment.Spec.Template.Spec.Affinity) {
		fmt.Printf("workload %s %s/%s already has arm preference, skip the prefer affinity\n",
			workload.Kind, workload.Namespace, workload.Name)
	} else {
		newDeployment.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
			AddArm64Preference(newDeployment.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	if CheckWorkloadHasARM64Toleration(newDeployment.Spec.Template.Spec.Tolerations) {
		fmt.Printf("workload %s %s/%s already has arm64 toleration, skip it\n",
			workload.Kind, workload.Namespace, workload.Name)
	} else {
		newDeployment.Spec.Template.Spec.Tolerations = AddARM64Toleration(newDeployment.Spec.Template.Spec.Tolerations)
	}

	return patchResource(ctx, deployment, newDeployment, workload.Namespace, workload.Name, workload.Kind)
}

func patchStatefulSetARMAffinity(workload *Workload) error {
	ctx := context.Background()

	ss, err := kubeClient.AppsV1().
		StatefulSets(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get statefulset: %w", err)
	}

	newSS := ss.DeepCopy()
	newSS.Spec.Template.Spec.Affinity = ensurePreferAffinity(newSS.Spec.Template.Spec.Affinity)

	if HasArm64Preference(newSS.Spec.Template.Spec.Affinity) {
		fmt.Printf("workload %s %s/%s already has arm preference, skip the prefer affinity\n",
			workload.Kind, workload.Namespace, workload.Name)
	} else {
		newSS.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
			AddArm64Preference(newSS.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}

	if CheckWorkloadHasARM64Toleration(newSS.Spec.Template.Spec.Tolerations) {
		fmt.Printf("workload %s %s/%s already has arm64 toleration, skip it\n",
			workload.Kind, workload.Namespace, workload.Name)
	} else {
		newSS.Spec.Template.Spec.Tolerations = AddARM64Toleration(newSS.Spec.Template.Spec.Tolerations)
	}

	return patchResource(ctx, ss, newSS, workload.Namespace, workload.Name, workload.Kind)
}
