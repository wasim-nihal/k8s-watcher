package config_test

import (
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
)

func BenchmarkConfig_UnmarshalSimple(b *testing.B) {
	yamlData := []byte(`
output:
  folder: /data
  uniqueFilenames: true
kubernetes:
  namespace: default
resources:
  type: configmap
  method: WATCH`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg config.Config
		_ = yaml.Unmarshal(yamlData, &cfg)
	}
}

func BenchmarkConfig_UnmarshalComplex(b *testing.B) {
	yamlData := []byte(`
output:
  folder: /data
  folderAnnotation: custom-annotation
  uniqueFilenames: true
  defaultFileMode: "440"
kubernetes:
  kubeconfig: /etc/kube/config
  namespace: default
  skipTLSVerify: true
resources:
  type: both
  method: WATCH
  resourceNames:
    - config1
    - config2
  watchConfig:
    serverTimeout: 120
    clientTimeout: 130
    errorThrottleTime: 10
    ignoreProcessed: true
  labels:
    - name: app
      value: myapp
      script:
        path: /scripts/notify.sh
        timeout: 30
      request:
        url: http://webhook.local
        method: POST
        timeout: 5.5
        retry:
          total: 3
          connect: 5
          read: 5
          backoffFactor: 2.0
        auth:
          basic:
            username: user
            password: pass
            encoding: utf8
logging:
  level: DEBUG
  format: JSON
  timezone: UTC`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var cfg config.Config
		_ = yaml.Unmarshal(yamlData, &cfg)
	}
}

func BenchmarkConfig_Marshal(b *testing.B) {
	cfg := config.Config{
		Output: config.OutputConfig{
			Folder:          "/data",
			UniqueFilenames: true,
		},
		Kubernetes: config.KubernetesConfig{
			Namespace: "default",
		},
		Resources: config.ResourceConfig{
			Type:   config.ResourceTypeConfigMap,
			Method: config.WatchMethodWatch,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = yaml.Marshal(cfg)
	}
}
