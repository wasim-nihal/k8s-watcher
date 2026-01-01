package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
	"github.com/wasim-nihal/k8s-watcher/pkg/version"
	"github.com/wasim-nihal/k8s-watcher/pkg/watcher"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		println(version.GetVersion())
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.NewLoader(*configPath).Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Initialize(cfg.Logging); err != nil {
		logger.Error("Failed to initialize logger", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting k8s-watcher", "version", version.GetVersion())

	// Create Kubernetes client
	client, err := createKubernetesClient(cfg.Kubernetes)
	if err != nil {
		logger.Error("Failed to create Kubernetes client", "error", err)
		os.Exit(1)
	}

	// Create and start watcher
	w := watcher.NewWatcher(client, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	logger.Info("Starting k8s-watcher")
	if err := w.Start(ctx); err != nil {
		logger.Error("Watcher failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Shutting down k8s-watcher")
}

// createKubernetesClient creates a Kubernetes client using the provided configuration
func createKubernetesClient(cfg config.KubernetesConfig) (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	if cfg.Kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, err
	}

	if cfg.SkipTLSVerify {
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
	}

	return kubernetes.NewForConfig(config)
}
