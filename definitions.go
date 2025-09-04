package main

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

type Workload struct {
	Name           string
	Namespace      string
	Kind           WorkloadKind
	Replicas       int32
	Available      int32
	Ready          bool
	MigratePatched bool
	ARMPatched     bool
	Priority       int32
	deployment     *appsv1.Deployment
	statefulSet    *appsv1.StatefulSet
}

type WorkloadKind string

const (
	WorkloadDeployment  WorkloadKind = "Deployment"
	WorkloadStatefulSet WorkloadKind = "StatefulSet"
)

var MigrateToleration = []corev1.Toleration{
	//{
	//	Key:      "cloudpilot.ai/provider-disable",
	//	Operator: corev1.TolerationOpEqual,
	//	Value:    "true",
	//	Effect:   corev1.TaintEffectNoSchedule,
	//},
	{
		Key:      "cloudpilot.ai/gradual-rebalance-only",
		Operator: corev1.TolerationOpExists,
	},
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
	if len(tolerations) == 0 {
		return false
	}

	for _, toleration := range tolerations {
		for _, migrateTol := range MigrateToleration {
			if toleration.Key == migrateTol.Key {
				return true
			}
		}
	}
	return false
}

func AddMigrateToleration(tolerations []corev1.Toleration) []corev1.Toleration {
	existingKeys := make(map[string]bool)
	for _, t := range tolerations {
		existingKeys[t.Key] = true
	}

	for _, migrateTol := range MigrateToleration {
		if !existingKeys[migrateTol.Key] {
			tolerations = append(tolerations, migrateTol)
		}
	}

	return tolerations
}

func RemoveMigrateToleration(tolerations []corev1.Toleration) []corev1.Toleration {
	migrateKeys := make(map[string]bool)
	for _, migrateTol := range MigrateToleration {
		migrateKeys[migrateTol.Key] = true
	}

	newTolerations := tolerations[:0]
	for _, t := range tolerations {
		if !migrateKeys[t.Key] {
			newTolerations = append(newTolerations, t)
		}
	}

	return newTolerations
}

var ArmPreferSchedulingTerm = corev1.PreferredSchedulingTerm{
	Weight: 10,
	Preference: corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      "kubernetes.io/arch",
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{"arm64"},
			},
		},
	},
}

func equalPref(a, b corev1.PreferredSchedulingTerm) bool {
	return equality.Semantic.DeepEqual(a.Preference, b.Preference)
}

func AddArm64Preference(existing []corev1.PreferredSchedulingTerm) []corev1.PreferredSchedulingTerm {
	for _, term := range existing {
		if equalPref(term, ArmPreferSchedulingTerm) {
			return existing
		}
	}
	return append(existing, ArmPreferSchedulingTerm)
}

func RemoveArm64Preference(existing []corev1.PreferredSchedulingTerm) []corev1.PreferredSchedulingTerm {
	filtered := existing[:0]
	for _, term := range existing {
		if !equalPref(term, ArmPreferSchedulingTerm) {
			filtered = append(filtered, term)
		}
	}
	return filtered
}

func HasArm64Preference(aff *corev1.Affinity) bool {
	tmp := aff.DeepCopy()
	tmp = ensurePreferAffinity(tmp)

	for _, term := range tmp.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
		if equalPref(term, ArmPreferSchedulingTerm) {
			return true
		}
	}
	return false
}

var ARM64Toleration = []corev1.Toleration{
	{
		Key:      "node.cloudpilot.ai/arch-arm64",
		Operator: corev1.TolerationOpExists,
	},
}

func CheckWorkloadHasARM64Toleration(tolerations []corev1.Toleration) bool {
	if len(tolerations) == 0 {
		return false
	}

	for _, toleration := range tolerations {
		for _, armTol := range ARM64Toleration {
			if toleration.Key == armTol.Key {
				return true
			}
		}
	}
	return false
}

func AddARM64Toleration(tolerations []corev1.Toleration) []corev1.Toleration {
	existingKeys := make(map[string]bool)
	for _, t := range tolerations {
		existingKeys[t.Key] = true
	}

	for _, armTol := range ARM64Toleration {
		if !existingKeys[armTol.Key] {
			tolerations = append(tolerations, armTol)
		}
	}

	return tolerations
}

func RemoveARM64Toleration(tolerations []corev1.Toleration) []corev1.Toleration {
	armKeys := make(map[string]bool)
	for _, armTol := range ARM64Toleration {
		armKeys[armTol.Key] = true
	}

	newTolerations := tolerations[:0]
	for _, t := range tolerations {
		if !armKeys[t.Key] {
			newTolerations = append(newTolerations, t)
		}
	}

	return newTolerations
}
