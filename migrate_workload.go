package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func migrateWorkload(selectedWorkloads []Workload) {
	for _, workload := range selectedWorkloads {
		var err error
		switch workload.Kind {
		case WorkloadDeployment:
			err = patchDeploymentMigrate(&workload)
		case WorkloadStatefulSet:
			err = patchStatefulSetMigrate(&workload)
		}
		if err != nil {
			fmt.Printf("failed to migrate %s workload %s/%s: %v\n", workload.Kind,
				workload.Namespace, workload.Name, err)
		}
	}
}

func patchDeploymentMigrate(workload *Workload) error {
	ctx := context.Background()
	deployment, err := kubeClient.AppsV1().Deployments(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newDeployment := deployment.DeepCopy()
	if newDeployment.Spec.Template.Spec.NodeSelector == nil {
		newDeployment.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	newDeployment.Spec.Template.Spec.NodeSelector[MigrateNodeSelectorKey] = MigrateNodeSelectorValue
	tolerationExists := CheckWorkloadHasMigrateToleration(deployment.Spec.Template.Spec.Tolerations)
	if !tolerationExists {
		newDeployment.Spec.Template.Spec.Tolerations = AddMigrateToleration(deployment.Spec.Template.Spec.Tolerations)
	}

	return patchResource(ctx, deployment, newDeployment, workload.Namespace, workload.Name, workload.Kind)
}

func patchStatefulSetMigrate(workload *Workload) error {
	ctx := context.Background()
	sts, err := kubeClient.AppsV1().StatefulSets(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newSts := sts.DeepCopy()
	if newSts.Spec.Template.Spec.NodeSelector == nil {
		newSts.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	newSts.Spec.Template.Spec.NodeSelector[MigrateNodeSelectorKey] = MigrateNodeSelectorValue
	tolerationExists := CheckWorkloadHasMigrateToleration(sts.Spec.Template.Spec.Tolerations)
	if !tolerationExists {
		newSts.Spec.Template.Spec.Tolerations = AddMigrateToleration(sts.Spec.Template.Spec.Tolerations)
	}

	return patchResource(ctx, sts, newSts, workload.Namespace, workload.Name, workload.Kind)
}
