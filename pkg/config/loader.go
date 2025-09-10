package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Loader handles configuration loading and validation
type Loader struct {
	path string
}

// NewLoader creates a new configuration loader
func NewLoader(path string) *Loader {
	return &Loader{path: path}
}

// Load reads and parses the configuration file
func (l *Loader) Load() (*Config, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := l.validate(&config); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	l.setDefaults(&config)
	return &config, nil
}

// validate checks the configuration for required fields and valid values
func (l *Loader) validate(cfg *Config) error {
	if cfg.Output.Folder == "" {
		return fmt.Errorf("output folder is required")
	}

	if err := l.validateResources(&cfg.Resources); err != nil {
		return err
	}

	return nil
}

// validateResources validates resource-specific configuration
func (l *Loader) validateResources(cfg *ResourceConfig) error {
	// Validate resource type
	switch cfg.Type {
	case ResourceTypeConfigMap, ResourceTypeSecret, ResourceTypeBoth:
		// Valid type
	default:
		return fmt.Errorf("invalid resource type: %s", cfg.Type)
	}

	// Validate watch method
	if cfg.Method != "" {
		switch cfg.Method {
		case WatchMethodWatch, WatchMethodList, WatchMethodSleep:
			// Valid method
		default:
			return fmt.Errorf("invalid watch method: %s", cfg.Method)
		}
	}

	// Validate labels
	if len(cfg.Labels) == 0 {
		return fmt.Errorf("at least one label configuration is required")
	}

	for i, label := range cfg.Labels {
		if label.Name == "" {
			return fmt.Errorf("label name is required for label config at index %d", i)
		}

		if err := l.validateRequest(label.Request); err != nil {
			return fmt.Errorf("invalid request config for label '%s': %w", label.Name, err)
		}

		if err := l.validateScript(label.Script); err != nil {
			return fmt.Errorf("invalid script config for label '%s': %w", label.Name, err)
		}
	}

	return nil
}

// validateRequest validates webhook request configuration
func (l *Loader) validateRequest(cfg RequestConfig) error {
	if cfg.URL != "" {
		if cfg.Method != "" && cfg.Method != "GET" && cfg.Method != "POST" {
			return fmt.Errorf("invalid request method: %s", cfg.Method)
		}

		if cfg.Timeout < 0 {
			return fmt.Errorf("timeout cannot be negative")
		}

		if cfg.Retry.Total < 0 || cfg.Retry.Connect < 0 || cfg.Retry.Read < 0 {
			return fmt.Errorf("retry counts cannot be negative")
		}

		if cfg.Retry.BackoffFactor < 1.0 {
			return fmt.Errorf("backoff factor must be greater than or equal to 1.0")
		}
	}

	return nil
}

// validateScript validates script execution configuration
func (l *Loader) validateScript(cfg ScriptConfig) error {
	if cfg.Path != "" {
		if cfg.Timeout < 0 {
			return fmt.Errorf("script timeout cannot be negative")
		}
	}

	return nil
}

// setDefaults sets default values for optional configuration fields
func (l *Loader) setDefaults(cfg *Config) {
	// Output defaults
	if cfg.Output.FolderAnnotation == "" {
		cfg.Output.FolderAnnotation = DefaultFolderAnnotation
	}

	// Watch config defaults
	if cfg.Resources.WatchConfig.ServerTimeout == 0 {
		cfg.Resources.WatchConfig.ServerTimeout = DefaultServerTimeout
	}
	if cfg.Resources.WatchConfig.ClientTimeout == 0 {
		cfg.Resources.WatchConfig.ClientTimeout = DefaultClientTimeout
	}
	if cfg.Resources.WatchConfig.ErrorThrottleTime == 0 {
		cfg.Resources.WatchConfig.ErrorThrottleTime = DefaultErrorThrottle
	}

	// Set defaults for each label config
	for i := range cfg.Resources.Labels {
		l.setLabelDefaults(&cfg.Resources.Labels[i])
	}

	// Logging defaults
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = DefaultLogLevel
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = DefaultLogFormat
	}
	if cfg.Logging.Timezone == "" {
		cfg.Logging.Timezone = DefaultLogTimezone
	}
}

// setLabelDefaults sets default values for label-specific configuration
func (l *Loader) setLabelDefaults(cfg *LabelConfig) {
	if cfg.Request.URL != "" {
		if cfg.Request.Method == "" {
			cfg.Request.Method = "GET"
		}
		if cfg.Request.Timeout == 0 {
			cfg.Request.Timeout = DefaultTimeout
		}
		if cfg.Request.Retry.Total == 0 {
			cfg.Request.Retry.Total = DefaultRetryTotal
		}
		if cfg.Request.Retry.Connect == 0 {
			cfg.Request.Retry.Connect = DefaultRetryConnect
		}
		if cfg.Request.Retry.Read == 0 {
			cfg.Request.Retry.Read = DefaultRetryRead
		}
		if cfg.Request.Retry.BackoffFactor == 0 {
			cfg.Request.Retry.BackoffFactor = DefaultBackoffFactor
		}
		if cfg.Request.Auth.Basic.Encoding == "" {
			cfg.Request.Auth.Basic.Encoding = DefaultAuthEncoding
		}
	}
}
