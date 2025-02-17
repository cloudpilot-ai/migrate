package main

import (
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type Workload struct {
	Name           string
	Namespace      string
	Kind           WorkloadKind
	Replicas       int32
	Available      int32
	Ready          bool
	MigratePatched bool
	deployment     *appsv1.Deployment
	statefulSet    *appsv1.StatefulSet
}

type WorkloadKind string

const (
	WorkloadDeployment  WorkloadKind = "Deployment"
	WorkloadStatefulSet WorkloadKind = "StatefulSet"
)

var MigrateToleration = corev1.Toleration{
	Key:      "cloudpilot.ai/provider-disable",
	Operator: corev1.TolerationOpEqual,
	Value:    "true",
	Effect:   corev1.TaintEffectNoSchedule,
}

var MigrateNodeSelectorKey = "node.cloudpilot.ai/managed"
var MigrateNodeSelectorValue = "true"

func CheckWorkloadIsMigrated(nodeSelector map[string]string, tolerations []corev1.Toleration) bool {
	if tolerations == nil || nodeSelector == nil {
		return false
	}

	return nodeSelector[MigrateNodeSelectorKey] == MigrateNodeSelectorValue &&
		CheckWorkloadHasMigrateToleration(tolerations)
}

func CheckWorkloadHasMigrateToleration(tolerations []corev1.Toleration) bool {
	if tolerations == nil || len(tolerations) == 0 {
		return false
	}

	for _, toleration := range tolerations {
		if reflect.DeepEqual(toleration, MigrateToleration) {
			return true
		}
	}
	return false
}
