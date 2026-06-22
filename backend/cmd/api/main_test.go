package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jmpsec/mapctf/pkg/config"
)

func TestLoadCheckConfigurationUsesCurrentConfigWhenNoFile(t *testing.T) {
	oldParams := flagParams
	t.Cleanup(func() { flagParams = oldParams })

	flagParams = config.ServiceFlagParams{
		ConfigFile: "/missing/mapctf.yaml",
		ConfigValues: config.MapCTFConfiguration{
			Service: config.ConfigurationService{
				Listener:  "127.0.0.1",
				Port:      "9000",
				LogLevel:  config.LogLevelInfo,
				LogFormat: config.LogFormatConsole,
				Auth:      config.AuthNone,
			},
			DB: config.ConfigurationDB{
				Type:     config.DBTypeSQLite,
				FilePath: ":memory:",
			},
			Redis: config.ConfigurationRedis{
				Host: "127.0.0.1",
				Port: "6379",
			},
		},
	}

	source, err := loadCheckConfiguration("")
	if err != nil {
		t.Fatal(err)
	}
	if source != "" {
		t.Fatalf("expected current configuration source, got %q", source)
	}
}

func TestLoadCheckConfigurationUsesProvidedFile(t *testing.T) {
	oldParams := flagParams
	t.Cleanup(func() { flagParams = oldParams })

	path := filepath.Join(t.TempDir(), "mapctf.yaml")
	if err := os.WriteFile(path, []byte(`service:
  listener: 127.0.0.1
  port: "9001"
  logLevel: info
  logFormat: console
  auth: none
db:
  type: sqlite
  filePath: ":memory:"
redis:
  host: 127.0.0.1
  port: "6379"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	source, err := loadCheckConfiguration(path)
	if err != nil {
		t.Fatal(err)
	}
	if source != path {
		t.Fatalf("expected source %q, got %q", path, source)
	}
	if flagParams.ConfigValues.Service.Port != "9001" {
		t.Fatalf("expected loaded port 9001, got %q", flagParams.ConfigValues.Service.Port)
	}
	if flagParams.ConfigValues.DB.Type != config.DBTypeSQLite {
		t.Fatalf("expected loaded sqlite db config, got %q", flagParams.ConfigValues.DB.Type)
	}
}
