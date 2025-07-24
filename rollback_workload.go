package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func rollbackWorkload(selectedWorkloads []Workload) {
	for _, workload := range selectedWorkloads {
		var err error
		switch workload.Kind {
		case WorkloadDeployment:
			err = rollbackDeploymentMigrate(&workload)
		case WorkloadStatefulSet:
			err = rollbackStatefulSetMigrate(&workload)
		}
		if err != nil {
			fmt.Printf("failed to rollback %s workload %s/%s, err: %v\n", workload.Kind,
				workload.Namespace, workload.Name, err)
		}
	}
}

func rollbackDeploymentMigrate(workload *Workload) error {
	ctx := context.Background()
	deployment, err := kubeClient.AppsV1().Deployments(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newDeployment := deployment.DeepCopy()
	if newDeployment.Spec.Template.Spec.NodeSelector != nil {
		delete(newDeployment.Spec.Template.Spec.NodeSelector, MigrateNodeSelectorKey)
	}
	tolerationExists := CheckWorkloadHasMigrateToleration(deployment.Spec.Template.Spec.Tolerations)
	if tolerationExists {
		newDeployment.Spec.Template.Spec.Tolerations = RemoveMigrateToleration(deployment.Spec.Template.Spec.Tolerations)
	}

	return patchResource(ctx, deployment, newDeployment, workload.Namespace, workload.Name, workload.Kind)
}

func rollbackStatefulSetMigrate(workload *Workload) error {
	ctx := context.Background()
	sts, err := kubeClient.AppsV1().StatefulSets(workload.Namespace).
		Get(ctx, workload.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	newSts := sts.DeepCopy()
	if newSts.Spec.Template.Spec.NodeSelector != nil {
		delete(newSts.Spec.Template.Spec.NodeSelector, MigrateNodeSelectorKey)
	}
	tolerationExists := CheckWorkloadHasMigrateToleration(sts.Spec.Template.Spec.Tolerations)
	if tolerationExists {
		newSts.Spec.Template.Spec.Tolerations = RemoveMigrateToleration(sts.Spec.Template.Spec.Tolerations)
	}

	return patchResource(ctx, sts, newSts, workload.Namespace, workload.Name, workload.Kind)
}
