package main

import (
	"flag"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func loadKubeClient() (*kubernetes.Clientset, error) {
	kubeconfig := flag.String("kubeconfig", "", "specified the path to the kubeconfig file")
	flag.Parse()
	if *kubeconfig == "" {
		return nil, fmt.Errorf("--kubeconfig is required")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
