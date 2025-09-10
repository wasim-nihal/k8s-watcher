package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wasim-nihal/k8s-watcher/pkg/config"
	"github.com/wasim-nihal/k8s-watcher/pkg/logger"
)

// Handler manages file operations for the watcher
type Handler struct {
	config config.OutputConfig
}

// NewHandler creates a new file handler
func NewHandler(config config.OutputConfig) *Handler {
	return &Handler{config: config}
}

// GetDefaultPath returns the default output path
func (h *Handler) GetDefaultPath() string {
	return h.config.Folder
}

// WriteFile writes data to a file with proper permissions
func (h *Handler) WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	mode, err := h.getFileMode()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}

	logger.Info("File written successfully",
		"path", path,
		"size", len(data),
		"mode", mode,
	)

	return nil
}

// DeleteFile removes a file from the filesystem
func (h *Handler) DeleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("deleting file %s: %w", path, err)
		}
		logger.Debug("File already deleted", "path", path)
		return nil
	}

	logger.Info("File deleted successfully", "path", path)
	return nil
}

// getFileMode returns the file mode from configuration
func (h *Handler) getFileMode() (os.FileMode, error) {
	if h.config.DefaultFileMode == "" {
		return 0644, nil
	}

	mode, err := strconv.ParseUint(h.config.DefaultFileMode, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("parsing file mode %s: %w", h.config.DefaultFileMode, err)
	}

	return os.FileMode(mode), nil
}

// GetOutputPath returns the final path for a resource file
func (h *Handler) GetOutputPath(name, namespace, key string) string {
	if h.config.UniqueFilenames {
		return filepath.Join(h.config.Folder, namespace, fmt.Sprintf("%s-%s", name, key))
	}
	return filepath.Join(h.config.Folder, namespace, name, key)
}

// GetAnnotationPath returns the path from a folder annotation
func (h *Handler) GetAnnotationPath(annotations map[string]string) string {
	if path, ok := annotations[h.config.FolderAnnotation]; ok {
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(h.config.Folder, path)
	}
	return h.config.Folder
}
