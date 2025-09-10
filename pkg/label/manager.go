package label

import (
	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// Manager handles label matching and configuration retrieval
type Manager struct {
	configs []config.LabelConfig
}

// NewManager creates a new label manager
func NewManager(configs []config.LabelConfig) *Manager {
	return &Manager{configs: configs}
}

// MatchLabels returns matching label configurations for the given resource labels
func (m *Manager) MatchLabels(resourceLabels map[string]string) []config.LabelConfig {
	var matches []config.LabelConfig

	for _, cfg := range m.configs {
		if m.matchLabel(cfg, resourceLabels) {
			matches = append(matches, cfg)
		}
	}

	return matches
}

// matchLabel checks if a single label configuration matches the resource labels
func (m *Manager) matchLabel(cfg config.LabelConfig, resourceLabels map[string]string) bool {
	if value, exists := resourceLabels[cfg.Name]; exists {
		if cfg.Value == "" || cfg.Value == value {
			return true
		}
	}
	return false
}

// GetSelector returns a label selector for all configured labels
func (m *Manager) GetSelector() (labels.Selector, error) {
	selector := labels.NewSelector()

	for _, cfg := range m.configs {
		if cfg.Value == "" {
			req, err := labels.NewRequirement(cfg.Name, selection.Exists, nil)
			if err != nil {
				return nil, err
			}
			selector = selector.Add(*req)
		} else {
			req, err := labels.NewRequirement(cfg.Name, selection.Equals, []string{cfg.Value})
			if err != nil {
				return nil, err
			}
			selector = selector.Add(*req)
		}
	}

	return selector, nil
}
