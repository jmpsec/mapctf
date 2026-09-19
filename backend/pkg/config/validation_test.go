package config

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v3"
)

func loadYAMLForTest(path string) (MapCTFConfiguration, error) {
	var cfg MapCTFConfiguration
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func TestValidateSchemaVersionAcceptsCurrentVersion(t *testing.T) {
	if err := ValidateSchemaVersion(ConfigSchemaVersion); err != nil {
		t.Fatalf("expected current version %d to be accepted, got error: %v", ConfigSchemaVersion, err)
	}
}

func TestValidateSchemaVersionAcceptsZeroForBackwardsCompatibility(t *testing.T) {
	if err := ValidateSchemaVersion(0); err != nil {
		t.Fatalf("expected version 0 (unversioned) to be accepted, got error: %v", err)
	}
}

func TestValidateSchemaVersionRejectsUnsupportedVersion(t *testing.T) {
	if err := ValidateSchemaVersion(99); err == nil {
		t.Fatal("expected error for unsupported version 99, got nil")
	}
}

func TestValidateSchemaVersionRejectsNegativeVersion(t *testing.T) {
	if err := ValidateSchemaVersion(-1); err == nil {
		t.Fatal("expected error for negative version -1, got nil")
	}
}

func TestValidateConfigValuesRejectsUnsupportedSchemaVersion(t *testing.T) {
	cfg := MapCTFConfiguration{
		SchemaVersion: 99,
		Service: ConfigurationService{
			Listener:  "0.0.0.0",
			Port:      "9000",
			LogLevel:  LogLevelInfo,
			LogFormat: LogFormatConsole,
			Auth:      AuthNone,
		},
		DB: ConfigurationDB{
			Type:     DBTypeSQLite,
			FilePath: ":memory:",
		},
		Redis: ConfigurationRedis{
			Host: "127.0.0.1",
			Port: "6379",
		},
	}
	if err := ValidateConfigValues(cfg); err == nil {
		t.Fatal("expected error for unsupported schema version in ValidateConfigValues, got nil")
	}
}

func TestValidateConfigValuesAcceptsUnversionedConfig(t *testing.T) {
	cfg := MapCTFConfiguration{
		SchemaVersion: 0,
		Service: ConfigurationService{
			Listener:  "0.0.0.0",
			Port:      "9000",
			LogLevel:  LogLevelInfo,
			LogFormat: LogFormatConsole,
			Auth:      AuthNone,
		},
		DB: ConfigurationDB{
			Type:     DBTypeSQLite,
			FilePath: ":memory:",
		},
		Redis: ConfigurationRedis{
			Host: "127.0.0.1",
			Port: "6379",
		},
	}
	if err := ValidateConfigValues(cfg); err != nil {
		t.Fatalf("expected unversioned config (version=0) to be accepted, got error: %v", err)
	}
}

func TestGenConfigFileSetsCurrentSchemaVersion(t *testing.T) {
	cfg := MapCTFConfiguration{
		SchemaVersion: 0,
		Service: ConfigurationService{
			Listener:  "0.0.0.0",
			Port:      "9000",
			LogLevel:  LogLevelInfo,
			LogFormat: LogFormatConsole,
			Auth:      AuthNone,
		},
		DB: ConfigurationDB{
			Type:     DBTypeSQLite,
			FilePath: ":memory:",
		},
		Redis: ConfigurationRedis{
			Host: "127.0.0.1",
			Port: "6379",
		},
	}
	path := t.TempDir() + "/test-config.yaml"
	if err := GenConfigFile(path, cfg, true); err != nil {
		t.Fatalf("GenConfigFile failed: %v", err)
	}
	// Read back and verify the version was stamped
	loaded, err := loadYAMLForTest(path)
	if err != nil {
		t.Fatalf("failed to load generated config: %v", err)
	}
	if loaded.SchemaVersion != ConfigSchemaVersion {
		t.Fatalf("expected generated config to have version %d, got %d", ConfigSchemaVersion, loaded.SchemaVersion)
	}
}
