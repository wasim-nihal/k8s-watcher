package main

import (
	"context"
	"flag"
	"fmt"
	watcher "k8s-cmwatcher-sidecar/app"
	"log"
	"os"
	"os/signal"
	"syscall"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	flag.Parse()
	err := watcher.InitConfig()
	if err != nil {
		log.Printf("unable to initialize watcher config. reason %s", err.Error())
		os.Exit(1)
	}
	watcher.Start()
	kubeconfigPath := "/home/wasim/.kube/config"

	// Load kubeconfig from the specified path
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Define ConfigMap watcher
	watcher, err := clientset.CoreV1().ConfigMaps("default").Watch(context.Background(), metav1.ListOptions{LabelSelector: "findme"})
	if err != nil {
		panic(err.Error())
	}
	defer watcher.Stop()

	// Handle signals to gracefully stop the watcher
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	// Continuous loop to watch for ConfigMap changes
	for {
		select {
		case event := <-watcher.ResultChan():
			switch event.Type {
			case watch.Added:
				fmt.Println("CM added")
			case watch.Modified:
				fmt.Println("CM modified")
			case watch.Deleted:
				fmt.Println("CM deleted")
			}
			// Handle ConfigMap change event
			cm, ok := event.Object.(*v1.ConfigMap)
			if !ok {
				continue
			}
			fmt.Printf("ConfigMap %s changed\n", cm.Name)
			// Process changes here
		case <-stopCh:
			// Stop the watcher on receiving termination signal
			fmt.Println("Received termination signal. Stopping watcher...")
			return
		}
	}
}
