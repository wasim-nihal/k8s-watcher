package watcher

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func Start() {
	cfg := GetWatcherConfig()
	if cfg.Resource.Type == "configmap" {
		for _, namespace := range strings.Split(cfg.Namespace, ",") {
			for idx, _ := range cfg.Resource.Labels {
				go watchCm(namespace, cfg, idx)
			}
		}
	}
}

func watchCm(namespace string, config WatcherConfig, idx int) {
	// Load kubeconfig from the specified path
	restConfig, err := clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err.Error())
	}
	watcher, err := clientset.CoreV1().ConfigMaps(config.Namespace).Watch(context.Background(), metav1.ListOptions{LabelSelector: getLabelSelector(config, idx)})
	if err != nil {
		panic(err.Error())
	}
	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			// Handle ConfigMap change event
			cm, ok := event.Object.(*v1.ConfigMap)
			if !ok {
				continue
			}
			switch event.Type {
			case watch.Added:
			case watch.Modified:
				log.Println("CM added or modified")
				ProcessAddUpdateCM(*cm, idx)
			case watch.Deleted:
				log.Println("CM deleted")
				ProcessDeleteCM(*cm, idx)
			}
		}
	}
}

func getLabelSelector(config WatcherConfig, idx int) string {
	selector := config.Resource.Labels[idx].Name
	if value := config.Resource.Labels[idx].Value; value != "" {
		selector += value
	}
	return selector
}
