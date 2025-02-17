package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
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
		newDeployment.Spec.Template.Spec.Tolerations = append(newDeployment.Spec.Template.Spec.Tolerations, MigrateToleration)
	}

	originalBytes, err := json.Marshal(deployment)
	if err != nil {
		return err
	}
	updatedBytes, err := json.Marshal(newDeployment)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalBytes, updatedBytes, appsv1.Deployment{})
	if err != nil {
		return err
	}

	err = backoff.Retry(func() error {
		_, patchErr := kubeClient.AppsV1().Deployments(workload.Namespace).
			Patch(ctx, workload.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		return patchErr
	}, DefaultBackoff(ctx))
	if err != nil {
		return fmt.Errorf("failed to patch Deployment: %v", err)
	}

	fmt.Printf("Patch deployment %s/%s successfully\n", workload.Namespace, workload.Name)
	return nil
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
		newSts.Spec.Template.Spec.Tolerations = append(newSts.Spec.Template.Spec.Tolerations, MigrateToleration)
	}

	originalBytes, err := json.Marshal(sts)
	if err != nil {
		return err
	}
	updatedBytes, err := json.Marshal(newSts)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalBytes, updatedBytes, appsv1.StatefulSet{})
	if err != nil {
		return err
	}

	err = backoff.Retry(func() error {
		_, patchErr := kubeClient.AppsV1().StatefulSets(workload.Namespace).
			Patch(ctx, workload.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
		return patchErr
	}, DefaultBackoff(ctx))
	if err != nil {
		return fmt.Errorf("failed to patch StatefulSet: %v", err)
	}

	fmt.Printf("Patch statefulset %s/%s successfully\n", workload.Namespace, workload.Name)
	return nil
}

func DefaultBackoff(ctx context.Context) backoff.BackOffContext {
	return backoff.WithContext(backoff.WithMaxRetries(backoff.NewConstantBackOff(1*time.Second), 5), ctx)
}
