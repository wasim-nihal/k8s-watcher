package config

import (
	"fmt"
)

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (r *ResourceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Create a temporary type to avoid infinite recursion
	type tempConfig ResourceConfig
	if err := unmarshal((*tempConfig)(r)); err != nil {
		return err
	}

	// Validate resource type
	switch r.Type {
	case ResourceTypeConfigMap, ResourceTypeSecret, ResourceTypeBoth:
		// Valid type
	default:
		return fmt.Errorf("invalid resource type: %s", r.Type)
	}

	// Set default values for WatchConfig
	if r.WatchConfig.ServerTimeout == 0 {
		r.WatchConfig.ServerTimeout = DefaultServerTimeout
	}
	if r.WatchConfig.ClientTimeout == 0 {
		r.WatchConfig.ClientTimeout = DefaultClientTimeout
	}
	if r.WatchConfig.ErrorThrottleTime == 0 {
		r.WatchConfig.ErrorThrottleTime = DefaultErrorThrottle
	}

	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (l *LoggingConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Create a temporary type to avoid infinite recursion
	type tempConfig LoggingConfig
	if err := unmarshal((*tempConfig)(l)); err != nil {
		return err
	}

	// Set default values
	if l.Level == "" {
		l.Level = DefaultLogLevel
	}
	if l.Format == "" {
		l.Format = DefaultLogFormat
	}
	if l.Timezone == "" {
		l.Timezone = DefaultLogTimezone
	}

	return nil
}
