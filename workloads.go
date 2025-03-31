package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func printWorkloadsTable(namespace string) error {
	var err error
	workloads, err = getAllWorkloads()
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"ID", "Namespace", "Kind", "Name", "Replicas", "Available", "Ready", "MigratePatched", "Priority"})

	for id, w := range workloads {
		if namespace == "" || namespace == w.Namespace {
			t.AppendRow(table.Row{
				id,
				w.Namespace,
				w.Kind,
				w.Name,
				w.Replicas,
				w.Available,
				func() interface{} {
					if w.Ready {
						return text.Colors{text.FgGreen}.Sprint("True")
					}
					return text.Colors{text.FgRed}.Sprint("False")
				}(),
				func() interface{} {
					if w.MigratePatched {
						return text.Colors{text.FgGreen}.Sprint("True")
					}
					return text.Colors{text.FgRed}.Sprint("False")
				}(),
				w.Priority,
			})
		}
	}

	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true, Colors: text.Colors{text.FgCyan}},
		{Number: 2, AutoMerge: true, Colors: text.Colors{text.FgCyan}},
	})
	t.Style().Options.SeparateRows = true

	t.Render()
	return nil
}

func getDeploymentsWorkloadPriority(deployment *appsv1.Deployment) int32 {
	// TODO: fix this
	// Get the priority of actual pods
	pods, err := kubeClient.CoreV1().Pods(deployment.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", deployment.Name),
	})
	if err != nil {
		klog.Errorf("Failed to get pods for deployment %s/%s, err: %v", deployment.Namespace, deployment.Name, err)
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
		klog.Errorf("Failed to get pods for statefulSet %s/%s, err: %v", statefulSet.Namespace, statefulSet.Name, err)
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
		klog.Errorf("Failed to get priority class %s, err: %v", pod.Spec.PriorityClassName, err)
		return 0
	}
	if priorityClass == nil {
		klog.Errorf("Priority class %s not found", pod.Spec.PriorityClassName)
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
			statefulSet:    &s,
			Priority:       priority,
		})
	}

	sort.Slice(newWorkloads, func(i, j int) bool {
		return newWorkloads[i].Namespace < newWorkloads[j].Namespace
	})
	return newWorkloads, nil
}
