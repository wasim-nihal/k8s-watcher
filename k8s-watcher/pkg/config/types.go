package config

// Config represents the root configuration structure
type Config struct {
	Output     OutputConfig     `yaml:"output"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	Resources  ResourceConfig   `yaml:"resources"`
	Logging    LoggingConfig    `yaml:"logging"`
}

// OutputConfig defines file output settings
type OutputConfig struct {
	Folder           string `yaml:"folder"`
	FolderAnnotation string `yaml:"folderAnnotation"`
	UniqueFilenames  bool   `yaml:"uniqueFilenames"`
	DefaultFileMode  string `yaml:"defaultFileMode"`
}

// KubernetesConfig defines Kubernetes connection settings
type KubernetesConfig struct {
	Kubeconfig    string `yaml:"kubeconfig"`
	Namespace     string `yaml:"namespace"`
	SkipTLSVerify bool   `yaml:"skipTLSVerify"`
}

// ResourceConfig defines resource watching configuration
type ResourceConfig struct {
	Type          string        `yaml:"type"`
	Method        string        `yaml:"method"`
	ResourceNames []string      `yaml:"resourceNames"`
	WatchConfig   WatchConfig   `yaml:"watchConfig"`
	Labels        []LabelConfig `yaml:"labels"`
}

// WatchConfig defines watch behavior settings
type WatchConfig struct {
	ServerTimeout     int  `yaml:"serverTimeout"`
	ClientTimeout     int  `yaml:"clientTimeout"`
	ErrorThrottleTime int  `yaml:"errorThrottleTime"`
	IgnoreProcessed   bool `yaml:"ignoreProcessed"`
}

// LabelConfig defines label selection and actions
type LabelConfig struct {
	Name    string        `yaml:"name"`
	Value   string        `yaml:"value"`
	Script  ScriptConfig  `yaml:"script"`
	Request RequestConfig `yaml:"request"`
}

// ScriptConfig defines script execution settings
type ScriptConfig struct {
	Path    string `yaml:"path"`
	Timeout int    `yaml:"timeout"`
}

// RequestConfig defines webhook configuration
type RequestConfig struct {
	URL           string      `yaml:"url"`
	Method        string      `yaml:"method"`
	Payload       interface{} `yaml:"payload"`
	Timeout       float64     `yaml:"timeout"`
	Retry         RetryConfig `yaml:"retry"`
	Auth          AuthConfig  `yaml:"auth"`
	SkipTLSVerify bool        `yaml:"skipTLSVerify"`
}

// RetryConfig defines retry behavior for HTTP requests
type RetryConfig struct {
	Total         int     `yaml:"total"`
	Connect       int     `yaml:"connect"`
	Read          int     `yaml:"read"`
	BackoffFactor float64 `yaml:"backoffFactor"`
}

// AuthConfig defines authentication settings
type AuthConfig struct {
	Basic        BasicAuth `yaml:"basic"`
	UsernameFile string    `yaml:"usernameFile"`
	PasswordFile string    `yaml:"passwordFile"`
}

// BasicAuth defines basic authentication credentials
type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Encoding string `yaml:"encoding"`
}

// LoggingConfig defines logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Timezone   string `yaml:"timezone"`
	ConfigPath string `yaml:"configPath"`
}

// Constants for configuration defaults and supported values
const (
	// Resource types
	ResourceTypeConfigMap = "configmap"
	ResourceTypeSecret    = "secret"
	ResourceTypeBoth      = "both"

	// Watch methods
	WatchMethodWatch = "WATCH"
	WatchMethodList  = "LIST"
	WatchMethodSleep = "SLEEP"

	// Default values
	DefaultFolderAnnotation = "k8s-sidecar-target-directory"
	DefaultServerTimeout    = 60
	DefaultClientTimeout    = 66
	DefaultErrorThrottle    = 5
	DefaultRetryTotal       = 5
	DefaultRetryConnect     = 10
	DefaultRetryRead        = 5
	DefaultBackoffFactor    = 1.1
	DefaultTimeout          = 10.0
	DefaultAuthEncoding     = "latin1"
	DefaultLogLevel         = "INFO"
	DefaultLogFormat        = "JSON"
	DefaultLogTimezone      = "LOCAL"
)
