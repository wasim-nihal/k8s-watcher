package label_test

import (
	"testing"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/label"
)

func TestManager(t *testing.T) {
	tests := []struct {
		name           string
		configs        []config.LabelConfig
		resourceLabels map[string]string
		want           []config.LabelConfig
	}{
		{
			name: "exact match",
			configs: []config.LabelConfig{
				{Name: "app", Value: "test"},
			},
			resourceLabels: map[string]string{
				"app": "test",
			},
			want: []config.LabelConfig{
				{Name: "app", Value: "test"},
			},
		},
		{
			name: "exists match",
			configs: []config.LabelConfig{
				{Name: "app", Value: ""},
			},
			resourceLabels: map[string]string{
				"app": "anyvalue",
			},
			want: []config.LabelConfig{
				{Name: "app", Value: ""},
			},
		},
		{
			name: "no match",
			configs: []config.LabelConfig{
				{Name: "app", Value: "test"},
			},
			resourceLabels: map[string]string{
				"app": "other",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := label.NewManager(tt.configs)
			got := manager.MatchLabels(tt.resourceLabels)
			matches := len(got) == len(tt.want)
			if matches {
				for i, cfg := range tt.want {
					if got[i].Name != cfg.Name || got[i].Value != cfg.Value {
						matches = false
						break
					}
				}
			}
			if !matches {
				t.Errorf("MatchLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSelector(t *testing.T) {
	tests := []struct {
		name      string
		configs   []config.LabelConfig
		wantError bool
	}{
		{
			name: "single label",
			configs: []config.LabelConfig{
				{Name: "app", Value: "test"},
			},
			wantError: false,
		},
		{
			name: "multiple labels",
			configs: []config.LabelConfig{
				{Name: "app", Value: "test"},
				{Name: "env", Value: "prod"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := label.NewManager(tt.configs)
			_, err := manager.GetSelector()
			if (err != nil) != tt.wantError {
				t.Errorf("GetSelector() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func BenchmarkMatchLabels(b *testing.B) {
	configs := []config.LabelConfig{
		{Name: "app", Value: "test"},
		{Name: "env", Value: "prod"},
	}
	manager := label.NewManager(configs)
	labels := map[string]string{
		"app": "test",
		"env": "prod",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.MatchLabels(labels)
	}
}
