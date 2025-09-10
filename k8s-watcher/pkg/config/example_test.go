package config_test

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
)

func ExampleConfig_basic() {
	yamlData := []byte(`
output:
  folder: /data
  uniqueFilenames: true
kubernetes:
  namespace: default
resources:
  type: configmap
  method: WATCH`)

	var cfg config.Config
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Output folder: %s\n", cfg.Output.Folder)
	fmt.Printf("Resource type: %s\n", cfg.Resources.Type)
	// Output:
	// Output folder: /data
	// Resource type: configmap
}

func ExampleConfig_withLabels() {
	yamlData := []byte(`
resources:
  type: configmap
  method: WATCH
  labels:
    - name: app
      value: myapp
      script:
        path: /scripts/notify.sh
      request:
        url: http://webhook.local`)

	var cfg config.Config
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	label := cfg.Resources.Labels[0]
	fmt.Printf("Label: %s=%s\n", label.Name, label.Value)
	fmt.Printf("Script path: %s\n", label.Script.Path)
	fmt.Printf("Webhook URL: %s\n", label.Request.URL)
	// Output:
	// Label: app=myapp
	// Script path: /scripts/notify.sh
	// Webhook URL: http://webhook.local
}
