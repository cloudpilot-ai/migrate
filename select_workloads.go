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
	if strings.Contains(input, ",") {
		workloadsIDs = strings.Split(input, ",")
	}
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

	printSelectedWorkloadsTable(selectedWorkloads)
	fmt.Print("Enter 'confirm' to confirm the workloads: ")

	if !scanner.Scan() {
		return nil, fmt.Errorf("input nothing")
	}
	confirmInput := scanner.Text()

	if confirmInput != "confirm" {
		return nil, fmt.Errorf("confirm failed")
	}

	return selectedWorkloads, nil
}

func printSelectedWorkloadsTable(selectedWorkloads []Workload) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"Namespace", "Kind", "Name", "Replicas", "Available", "Ready", "MigratePatched"})

	for _, w := range selectedWorkloads {
		t.AppendRow(table.Row{
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

	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true, Colors: text.Colors{text.FgCyan}},
	})
	t.Style().Options.SeparateRows = true

	t.Render()
	return
}
