package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func selectWorkloads(scanner *bufio.Scanner) ([]Workload, error) {
	fmt.Println("Input the workload id which should be selected, example: 0,1,2,3; or a range, example 1-9")
	fmt.Print("Input: ")

	if !scanner.Scan() {
		return nil, fmt.Errorf("input nothing")
	}
	input := scanner.Text()

	var workloadsIDs []string
	if strings.Contains(input, "-") {
		ids := strings.Split(input, "-")
		startIDStr, endIDStr := ids[0], ids[1]

		startID, err := strconv.Atoi(startIDStr)
		if err != nil {
			return nil, err
		}
		endID, err := strconv.Atoi(endIDStr)
		if err != nil {
			return nil, err
		}
		for i := startID; i <= endID; i++ {
			workloadsIDs = append(workloadsIDs, strconv.Itoa(i))
		}
	} else {
		workloadsIDs = strings.Split(input, ",")
	}

	selectedWorkloads := make([]Workload, len(workloadsIDs))
	for i, id := range workloadsIDs {
		idInt, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("error converting workload id '%s' to int", id)
		}
		if idInt >= len(workloads) {
			return nil, fmt.Errorf("wrong workload id '%s'", id)
		}
		selectedWorkloads[i] = workloads[idInt]
	}

	printSelectedWorkloadsTable(selectedWorkloads, "")
	fmt.Print("Press 'Enter' to confirm the workloads, or input others to skip: ")

	if !scanner.Scan() {
		return nil, fmt.Errorf("input nothing")
	}
	confirmInput := scanner.Text()

	if confirmInput != "" {
		fmt.Println("You have skipped the workloads.")
		return nil, fmt.Errorf("confirm failed")
	}

	return selectedWorkloads, nil
}

func printSelectedWorkloadsTable(selectedWorkloads []Workload, namespace string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	armSupported := CheckAllWorkloadsArm(selectedWorkloads)

	t.AppendHeader(table.Row{"ID", "Namespace", "Kind", "Name", "Replicas", "Available", "Ready",
		"MigratePatched", "ARMSupported", "ARMPatched", "Priority"})

	for id, w := range selectedWorkloads {
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
				func() interface{} {
					if armSupported[id].Err != nil {
						fmt.Printf("Failed to check arm support for workload %s %s/%s: %v",
							w.Kind, w.Namespace, w.Name, armSupported[id].Err)
						return text.Colors{text.FgRed}.Sprint("Unknown")
					}
					if armSupported[id].Supported {
						return text.Colors{text.FgGreen}.Sprint("True")
					}
					return text.Colors{text.FgRed}.Sprint("False")
				}(),
				func() interface{} {
					if w.ARMPatched {
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
		{Number: 3, AutoMerge: true, Colors: text.Colors{text.FgCyan}},
	})
	t.Style().Options.SeparateRows = true

	t.Render()
	return
}
