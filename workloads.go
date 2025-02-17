package main

import (
	"context"
	"os"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	t.AppendHeader(table.Row{"ID", "Namespace", "Kind", "Name", "Replicas", "Available", "Ready", "MigratePatched"})

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

func getAllWorkloads() ([]Workload, error) {
	var newWorkloads []Workload

	deployments, err := kubeClient.AppsV1().Deployments(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range deployments.Items {
		newWorkloads = append(newWorkloads, Workload{
			Name:           d.Name,
			Namespace:      d.Namespace,
			Kind:           WorkloadDeployment,
			Replicas:       *d.Spec.Replicas,
			Available:      d.Status.AvailableReplicas,
			Ready:          CheckDeploymentIsReady(&d),
			MigratePatched: CheckWorkloadIsMigrated(d.Spec.Template.Spec.NodeSelector, d.Spec.Template.Spec.Tolerations),
			deployment:     &d,
		})
	}

	statefulSets, err := kubeClient.AppsV1().StatefulSets(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, s := range statefulSets.Items {
		newWorkloads = append(newWorkloads, Workload{
			Name:           s.Name,
			Namespace:      s.Namespace,
			Kind:           WorkloadStatefulSet,
			Replicas:       *s.Spec.Replicas,
			Available:      s.Status.ReadyReplicas,
			Ready:          CheckStatefulSetIsReady(&s),
			MigratePatched: CheckWorkloadIsMigrated(s.Spec.Template.Spec.NodeSelector, s.Spec.Template.Spec.Tolerations),
			statefulSet:    &s,
		})
	}

	sort.Slice(newWorkloads, func(i, j int) bool {
		return newWorkloads[i].Namespace < newWorkloads[j].Namespace
	})
	return newWorkloads, nil
}
