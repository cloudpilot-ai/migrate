package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func CheckStatefulSetIsReady(sts *appsv1.StatefulSet) bool {
	return sts.Status.ObservedGeneration == sts.Generation &&
		sts.Status.Replicas == *sts.Spec.Replicas &&
		sts.Status.ReadyReplicas == *sts.Spec.Replicas &&
		sts.Status.CurrentReplicas == *sts.Spec.Replicas
}

func CheckDeploymentIsReady(deployment *appsv1.Deployment) bool {
	replicaFailure := false
	progressing := false
	available := false

	for _, condition := range deployment.Status.Conditions {
		switch condition.Type {
		case appsv1.DeploymentProgressing:
			if condition.Status == corev1.ConditionTrue && condition.Reason == "NewReplicaSetAvailable" {
				progressing = true
			}
		case appsv1.DeploymentAvailable:
			if condition.Status == corev1.ConditionTrue {
				available = true
			}
		case appsv1.DeploymentReplicaFailure:
			if condition.Status == corev1.ConditionTrue {
				replicaFailure = true
				break
			}
		}
	}

	return deployment.Status.ObservedGeneration == deployment.Generation &&
		deployment.Status.Replicas == *deployment.Spec.Replicas &&
		deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&
		deployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
		deployment.Status.Conditions != nil && len(deployment.Status.Conditions) > 0 &&
		(progressing || available) && !replicaFailure
}
