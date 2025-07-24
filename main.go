package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"k8s.io/client-go/kubernetes"
)

var kubeClient *kubernetes.Clientset

var workloads []Workload

func main() {
	var err error
	kubeClient, err = loadKubeClient()
	if err != nil {
		log.Fatalf("Failed to create kubernetes client, err: %v", err)
	}

	if err := printWorkloadsTable(""); err != nil {
		log.Fatalf("Failed to print workloads table, err: %v", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\nPlease choose a action:")
		fmt.Println("1. Show all workloads")
		fmt.Println("2. Migrate workload")
		fmt.Println("3. Rollback workload")
		fmt.Println("4. Patch workload ARM affinity")
		fmt.Println("5. Rollback workload ARM affinity")
		fmt.Println("6. Exit")
		fmt.Print("Input the action number: ")

		if !scanner.Scan() {
			continue
		}
		choice := scanner.Text()

		switch choice {
		case "1":
			fmt.Print("Please enter a namespace (leave empty to show all workloads):")
			scanner.Scan()
			if err := printWorkloadsTable(scanner.Text()); err != nil {
				log.Printf("Failed to print workloads table, err: %v\n", err)
			}
		case "2":
			selectedWorkloads, err := selectWorkloads(scanner)
			if err != nil {
				log.Printf("Failed to select workloads, err: %v\n", err)
			}
			migrateWorkload(selectedWorkloads)
		case "3":
			selectedWorkloads, err := selectWorkloads(scanner)
			if err != nil {
				log.Printf("Failed to select workloads, err: %v\n", err)
			}
			rollbackWorkload(selectedWorkloads)
		case "4":
			selectedWorkloads, err := selectWorkloads(scanner)
			if err != nil {
				log.Printf("Failed to select workloads, err: %v\n", err)
			}
			patchWorkloadARMAffinity(selectedWorkloads)
		case "5":
			selectedWorkloads, err := selectWorkloads(scanner)
			if err != nil {
				log.Printf("Failed to select workloads, err: %v\n", err)
			}
			rollbackWorkloadARMAffinity(selectedWorkloads)
		case "6":
			return
		}
	}
}
