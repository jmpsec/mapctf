package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

// Helper to generate an example configuration file
func GenConfigFile(path string, cfg MapCTFConfiguration, overwrite bool) error {
	cfg.ServiceConfigFile = ""
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	if path == "" {
		return fmt.Errorf("output path cannot be empty")
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file %s already exists (use --force to overwrite)", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to check if %s exists: %w", path, err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write configuration to %s: %w", path, err)
	}
	return nil
}

// Helper to generate a UUID
func GenUUID() string {
	return uuid.New().String()
}
