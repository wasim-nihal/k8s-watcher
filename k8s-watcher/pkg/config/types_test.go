package config

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestConfigUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    Config
		wantErr bool
	}{
		{
			name: "basic config",
			yaml: `
output:
  folder: /data
  uniqueFilenames: true
kubernetes:
  namespace: default
resources:
  type: configmap
  method: WATCH`,
			want: Config{
				Output: OutputConfig{
					Folder:          "/data",
					UniqueFilenames: true,
				},
				Kubernetes: KubernetesConfig{
					Namespace: "default",
				},
				Resources: ResourceConfig{
					Type:   ResourceTypeConfigMap,
					Method: WatchMethodWatch,
					WatchConfig: WatchConfig{
						ServerTimeout:     DefaultServerTimeout,
						ClientTimeout:     DefaultClientTimeout,
						ErrorThrottleTime: DefaultErrorThrottle,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "full config",
			yaml: `
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
  timezone: UTC`,
			want: Config{
				Output: OutputConfig{
					Folder:           "/data",
					FolderAnnotation: "custom-annotation",
					UniqueFilenames:  true,
					DefaultFileMode:  "440",
				},
				Kubernetes: KubernetesConfig{
					Kubeconfig:    "/etc/kube/config",
					Namespace:     "default",
					SkipTLSVerify: true,
				},
				Resources: ResourceConfig{
					Type:          ResourceTypeBoth,
					Method:        WatchMethodWatch,
					ResourceNames: []string{"config1", "config2"},
					WatchConfig: WatchConfig{
						ServerTimeout:     120,
						ClientTimeout:     130,
						ErrorThrottleTime: 10,
						IgnoreProcessed:   true,
					},
					Labels: []LabelConfig{
						{
							Name:  "app",
							Value: "myapp",
							Script: ScriptConfig{
								Path:    "/scripts/notify.sh",
								Timeout: 30,
							},
							Request: RequestConfig{
								URL:     "http://webhook.local",
								Method:  "POST",
								Timeout: 5.5,
								Retry: RetryConfig{
									Total:         3,
									Connect:       5,
									Read:          5,
									BackoffFactor: 2.0,
								},
								Auth: AuthConfig{
									Basic: BasicAuth{
										Username: "user",
										Password: "pass",
										Encoding: "utf8",
									},
								},
							},
						},
					},
				},
				Logging: LoggingConfig{
					Level:    "DEBUG",
					Format:   "JSON",
					Timezone: "UTC",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid resource type",
			yaml: `
output:
  folder: /data
resources:
  type: invalid
  method: WATCH`,
			wantErr: true,
		},
		{
			name: "empty resource type",
			yaml: `
output:
  folder: /data
resources:
  method: WATCH`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Config
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("yaml.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !configEquals(got, tt.want) {
				t.Errorf("yaml.Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want Config
	}{
		{
			name: "default watch config values",
			yaml: `
resources:
  type: configmap
  method: WATCH`,
			want: Config{
				Resources: ResourceConfig{
					Type:   ResourceTypeConfigMap,
					Method: WatchMethodWatch,
					WatchConfig: WatchConfig{
						ServerTimeout:     DefaultServerTimeout,
						ClientTimeout:     DefaultClientTimeout,
						ErrorThrottleTime: DefaultErrorThrottle,
					},
				},
			},
		},
		{
			name: "default logging values",
			yaml: `
logging: {}`,
			want: Config{
				Logging: LoggingConfig{
					Level:    DefaultLogLevel,
					Format:   DefaultLogFormat,
					Timezone: DefaultLogTimezone,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Config
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			if err != nil {
				t.Errorf("yaml.Unmarshal() error = %v", err)
				return
			}
			if !configEquals(got, tt.want) {
				t.Errorf("yaml.Unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare Config structs
func configEquals(a, b Config) bool {
	// Compare OutputConfig
	if a.Output.Folder != b.Output.Folder ||
		a.Output.FolderAnnotation != b.Output.FolderAnnotation ||
		a.Output.UniqueFilenames != b.Output.UniqueFilenames ||
		a.Output.DefaultFileMode != b.Output.DefaultFileMode {
		return false
	}

	// Compare KubernetesConfig
	if a.Kubernetes.Kubeconfig != b.Kubernetes.Kubeconfig ||
		a.Kubernetes.Namespace != b.Kubernetes.Namespace ||
		a.Kubernetes.SkipTLSVerify != b.Kubernetes.SkipTLSVerify {
		return false
	}

	// Compare ResourceConfig
	if a.Resources.Type != b.Resources.Type ||
		a.Resources.Method != b.Resources.Method ||
		!stringSliceEqual(a.Resources.ResourceNames, b.Resources.ResourceNames) {
		return false
	}

	// Compare WatchConfig
	if a.Resources.WatchConfig.ServerTimeout != b.Resources.WatchConfig.ServerTimeout ||
		a.Resources.WatchConfig.ClientTimeout != b.Resources.WatchConfig.ClientTimeout ||
		a.Resources.WatchConfig.ErrorThrottleTime != b.Resources.WatchConfig.ErrorThrottleTime ||
		a.Resources.WatchConfig.IgnoreProcessed != b.Resources.WatchConfig.IgnoreProcessed {
		return false
	}

	// Compare LoggingConfig
	if a.Logging.Level != b.Logging.Level ||
		a.Logging.Format != b.Logging.Format ||
		a.Logging.Timezone != b.Logging.Timezone ||
		a.Logging.ConfigPath != b.Logging.ConfigPath {
		return false
	}

	return true
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func BenchmarkConfigUnmarshal(b *testing.B) {
	yamlData := []byte(`
output:
  folder: /data
  uniqueFilenames: true
kubernetes:
  namespace: default
resources:
  type: configmap
  method: WATCH
  watchConfig:
    serverTimeout: 120
    clientTimeout: 130
    errorThrottleTime: 10
  labels:
    - name: app
      value: myapp`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var c Config
		_ = yaml.Unmarshal(yamlData, &c)
	}
}
