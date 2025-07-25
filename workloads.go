package main

import (
	"context"
	"fmt"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func printWorkloadsTable(namespace string) error {
	var err error
	workloads, err = getAllWorkloads()
	if err != nil {
		return err
	}

	printSelectedWorkloadsTable(workloads, namespace)
	return nil
}

func getDeploymentsWorkloadPriority(deployment *appsv1.Deployment) int32 {
	// Get the priority of actual pods
	pods, err := kubeClient.CoreV1().Pods(deployment.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", deployment.Name),
	})
	if err != nil {
		fmt.Printf("Failed to get pods for deployment %s/%s, err: %v", deployment.Namespace, deployment.Name, err)
		return 0
	}
	if len(pods.Items) == 0 {
		return 0
	}

	// Get the priority of the first pod, we do not support multiple priority in one deployment now
	return getPodPriority(&pods.Items[0])
}

func getStatefulSetWorkloadPriority(statefulSet *appsv1.StatefulSet) int32 {
	// Get the priority of actual pods
	pods, err := kubeClient.CoreV1().Pods(statefulSet.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", statefulSet.Name),
	})
	if err != nil {
		fmt.Printf("Failed to get pods for statefulSet %s/%s, err: %v", statefulSet.Namespace, statefulSet.Name, err)
		return 0
	}
	if len(pods.Items) == 0 {
		return 0
	}

	// Get the priority of the first pod, we do not support multiple priority in one deployment now
	return getPodPriority(&pods.Items[0])
}

func getPodPriority(pod *corev1.Pod) int32 {
	if pod.Spec.PriorityClassName == "" {
		return 0
	}

	priorityClass, err := kubeClient.SchedulingV1().PriorityClasses().Get(context.TODO(), pod.Spec.PriorityClassName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("Failed to get priority class %s, err: %v", pod.Spec.PriorityClassName, err)
		return 0
	}
	if priorityClass == nil {
		fmt.Printf("Priority class %s not found", pod.Spec.PriorityClassName)
		return 0
	}

	return priorityClass.Value
}

func getAllWorkloads() ([]Workload, error) {
	var newWorkloads []Workload

	deployments, err := kubeClient.AppsV1().Deployments(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range deployments.Items {
		// Get the priority of the deployment
		priority := getDeploymentsWorkloadPriority(&d)
		newWorkloads = append(newWorkloads, Workload{
			Name:           d.Name,
			Namespace:      d.Namespace,
			Kind:           WorkloadDeployment,
			Replicas:       *d.Spec.Replicas,
			Available:      d.Status.AvailableReplicas,
			Ready:          CheckDeploymentIsReady(&d),
			MigratePatched: CheckWorkloadIsMigrated(d.Spec.Template.Spec.NodeSelector, d.Spec.Template.Spec.Tolerations),
			ARMPatched:     HasArm64Preference(d.Spec.Template.Spec.Affinity) || CheckWorkloadHasARM64Toleration(d.Spec.Template.Spec.Tolerations),
			deployment:     &d,
			Priority:       priority,
		})
	}

	statefulSets, err := kubeClient.AppsV1().StatefulSets(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, s := range statefulSets.Items {
		// Get the priority of the statefulSet
		priority := getStatefulSetWorkloadPriority(&s)
		newWorkloads = append(newWorkloads, Workload{
			Name:           s.Name,
			Namespace:      s.Namespace,
			Kind:           WorkloadStatefulSet,
			Replicas:       *s.Spec.Replicas,
			Available:      s.Status.ReadyReplicas,
			Ready:          CheckStatefulSetIsReady(&s),
			MigratePatched: CheckWorkloadIsMigrated(s.Spec.Template.Spec.NodeSelector, s.Spec.Template.Spec.Tolerations),
			ARMPatched:     HasArm64Preference(s.Spec.Template.Spec.Affinity) || CheckWorkloadHasARM64Toleration(s.Spec.Template.Spec.Tolerations),
			statefulSet:    &s,
			Priority:       priority,
		})
	}

	sort.Slice(newWorkloads, func(i, j int) bool {
		wi, wj := newWorkloads[i], newWorkloads[j]
		// Sort by priority first
		if wi.Namespace != wj.Namespace {
			return wi.Namespace < wj.Namespace
		}
		// Then by workload kind
		if wi.Kind != wj.Kind {
			return wi.Kind < wj.Kind
		}
		return wi.Name < wj.Name
	})
	return newWorkloads, nil
}
