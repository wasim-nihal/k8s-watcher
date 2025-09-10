package file_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/file"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

func TestNewHandler(t *testing.T) {
	cfg := config.OutputConfig{
		Folder:          "/tmp/test",
		UniqueFilenames: true,
		DefaultFileMode: "644",
	}

	handler := file.NewHandler(cfg)
	if handler == nil {
		t.Error("NewHandler returned nil")
	}
}

func TestGetDefaultPath(t *testing.T) {
	tests := []struct {
		name   string
		config config.OutputConfig
		want   string
	}{
		{
			name: "basic path",
			config: config.OutputConfig{
				Folder: "/tmp/test",
			},
			want: "/tmp/test",
		},
		{
			name: "empty path",
			config: config.OutputConfig{
				Folder: "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := file.NewHandler(tt.config)
			got := handler.GetDefaultPath()
			if got != tt.want {
				t.Errorf("GetDefaultPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	// Initialize logger
	err := logger.Initialize(config.LoggingConfig{
		Level:  "INFO",
		Format: "JSON",
	})
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "file-handler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name       string
		config     config.OutputConfig
		path       string
		data       []byte
		fileMode   string
		wantErr    bool
		wantExists bool
	}{
		{
			name: "basic write",
			config: config.OutputConfig{
				Folder:          tempDir,
				DefaultFileMode: "644",
			},
			path:       filepath.Join(tempDir, "test.txt"),
			data:       []byte("test data"),
			wantErr:    false,
			wantExists: true,
		},
		{
			name: "write with custom mode",
			config: config.OutputConfig{
				Folder:          tempDir,
				DefaultFileMode: "600",
			},
			path:       filepath.Join(tempDir, "secure.txt"),
			data:       []byte("secure data"),
			wantErr:    false,
			wantExists: true,
		},
		{
			name: "invalid file mode",
			config: config.OutputConfig{
				Folder:          tempDir,
				DefaultFileMode: "999",
			},
			path:       filepath.Join(tempDir, "invalid.txt"),
			data:       []byte("test data"),
			wantErr:    true,
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := file.NewHandler(tt.config)
			err := handler.WriteFile(tt.path, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			exists := fileExists(tt.path)
			if exists != tt.wantExists {
				t.Errorf("File exists = %v, want %v", exists, tt.wantExists)
				return
			}

			if exists {
				gotData, err := os.ReadFile(tt.path)
				if err != nil {
					t.Errorf("Failed to read written file: %v", err)
					return
				}
				if string(gotData) != string(tt.data) {
					t.Errorf("File content = %v, want %v", string(gotData), string(tt.data))
				}
			}
		})
	}
}

func TestDeleteFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "file-handler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name          string
		config        config.OutputConfig
		setupFile     bool
		path          string
		wantErr       bool
		wantExistsEnd bool
	}{
		{
			name: "delete existing file",
			config: config.OutputConfig{
				Folder: tempDir,
			},
			setupFile:     true,
			path:          filepath.Join(tempDir, "exists.txt"),
			wantErr:       false,
			wantExistsEnd: false,
		},
		{
			name: "delete non-existent file",
			config: config.OutputConfig{
				Folder: tempDir,
			},
			setupFile:     false,
			path:          filepath.Join(tempDir, "nonexistent.txt"),
			wantErr:       false,
			wantExistsEnd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := file.NewHandler(tt.config)

			if tt.setupFile {
				if err := os.WriteFile(tt.path, []byte("test"), 0644); err != nil {
					t.Fatalf("Failed to setup test file: %v", err)
				}
			}

			err := handler.DeleteFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			exists := fileExists(tt.path)
			if exists != tt.wantExistsEnd {
				t.Errorf("File exists = %v, want %v", exists, tt.wantExistsEnd)
			}
		})
	}
}

func TestGetOutputPath(t *testing.T) {
	tests := []struct {
		name         string
		config       config.OutputConfig
		resourceName string
		namespace    string
		key          string
		want         string
	}{
		{
			name: "unique filenames enabled",
			config: config.OutputConfig{
				Folder:          "/data",
				UniqueFilenames: true,
			},
			resourceName: "myconfig",
			namespace:    "default",
			key:          "config.yaml",
			want:         "/data/default/myconfig-config.yaml",
		},
		{
			name: "unique filenames disabled",
			config: config.OutputConfig{
				Folder:          "/data",
				UniqueFilenames: false,
			},
			resourceName: "myconfig",
			namespace:    "default",
			key:          "config.yaml",
			want:         "/data/default/myconfig/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := file.NewHandler(tt.config)
			got := handler.GetOutputPath(tt.resourceName, tt.namespace, tt.key)
			if got != tt.want {
				t.Errorf("GetOutputPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAnnotationPath(t *testing.T) {
	tests := []struct {
		name        string
		config      config.OutputConfig
		annotations map[string]string
		want        string
	}{
		{
			name: "with annotation absolute path",
			config: config.OutputConfig{
				Folder:           "/data",
				FolderAnnotation: "target-folder",
			},
			annotations: map[string]string{
				"target-folder": "/custom/path",
			},
			want: "/custom/path",
		},
		{
			name: "with annotation relative path",
			config: config.OutputConfig{
				Folder:           "/data",
				FolderAnnotation: "target-folder",
			},
			annotations: map[string]string{
				"target-folder": "custom/path",
			},
			want: "/data/custom/path",
		},
		{
			name: "without annotation",
			config: config.OutputConfig{
				Folder:           "/data",
				FolderAnnotation: "target-folder",
			},
			annotations: map[string]string{},
			want:        "/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := file.NewHandler(tt.config)
			got := handler.GetAnnotationPath(tt.annotations)
			if got != tt.want {
				t.Errorf("GetAnnotationPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkWriteFile(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "file-handler-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	handler := file.NewHandler(config.OutputConfig{
		Folder:          tempDir,
		DefaultFileMode: "644",
	})

	data := []byte("benchmark test data")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("bench-%d.txt", i))
		if err := handler.WriteFile(path, data); err != nil {
			b.Fatalf("WriteFile failed: %v", err)
		}
	}
}

func BenchmarkGetOutputPath(b *testing.B) {
	handler := file.NewHandler(config.OutputConfig{
		Folder:          "/data",
		UniqueFilenames: true,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.GetOutputPath("resource", "namespace", "key")
	}
}

// Helper function to check if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
