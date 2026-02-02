package backend

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jmpsec/mapctf/pkg/config"
)

// TestPrepareDSNPostgres tests DSN preparation for PostgreSQL
func TestPrepareDSNPostgres(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     DBTypePostgres,
		Host:     "localhost",
		Port:     "5432",
		Name:     "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	dsn := PrepareDSN(cfg)
	expected := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"

	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

// TestPrepareDSNMySQL tests DSN preparation for MySQL
func TestPrepareDSNMySQL(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     DBTypeMySQL,
		Host:     "localhost",
		Port:     "3306",
		Name:     "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	dsn := PrepareDSN(cfg)
	expected := "testuser:testpass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"

	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

// TestPrepareDSNSQLite tests DSN preparation for SQLite
func TestPrepareDSNSQLite(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     DBTypeSQLite,
		FilePath: "/tmp/test.db",
	}

	dsn := PrepareDSN(cfg)
	expected := "/tmp/test.db"

	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

// TestPrepareDSNDefault tests DSN preparation with unknown type defaults to Postgres
func TestPrepareDSNDefault(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     "unknown",
		Host:     "localhost",
		Port:     "5432",
		Name:     "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	dsn := PrepareDSN(cfg)
	expected := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=disable"

	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

// TestPrepareDSNWithSpecialCharacters tests DSN with special characters
func TestPrepareDSNWithSpecialCharacters(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     DBTypePostgres,
		Host:     "db.example.com",
		Port:     "5433",
		Name:     "my-database",
		Username: "user@domain",
		Password: "p@ssw0rd!",
		SSLMode:  "require",
	}

	dsn := PrepareDSN(cfg)
	expected := "host=db.example.com port=5433 dbname=my-database user=user@domain password=p@ssw0rd! sslmode=require"

	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

// TestLoadConfigurationSuccess tests successful configuration loading
func TestLoadConfigurationSuccess(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `db:
  type: sqlite
  filePath: /tmp/test.db
  maxIdleConns: 5
  maxOpenConns: 10
  connMaxLifetime: 300
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadConfiguration(configFile, DBKey)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.Type != DBTypeSQLite {
		t.Errorf("Expected type %q, got %q", DBTypeSQLite, cfg.Type)
	}

	if cfg.FilePath != "/tmp/test.db" {
		t.Errorf("Expected filePath %q, got %q", "/tmp/test.db", cfg.FilePath)
	}

	if cfg.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns 5, got %d", cfg.MaxIdleConns)
	}

	if cfg.MaxOpenConns != 10 {
		t.Errorf("Expected MaxOpenConns 10, got %d", cfg.MaxOpenConns)
	}

	if cfg.ConnMaxLifetime != 300 {
		t.Errorf("Expected ConnMaxLifetime 300, got %d", cfg.ConnMaxLifetime)
	}
}

// TestLoadConfigurationFileNotFound tests error handling when file doesn't exist
func TestLoadConfigurationFileNotFound(t *testing.T) {
	_, err := LoadConfiguration("/nonexistent/path/config.yaml", DBKey)
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}

// TestLoadConfigurationInvalidKey tests error handling when key is not found
func TestLoadConfigurationInvalidKey(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `service:
  port: 8080
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := LoadConfiguration(configFile, DBKey)
	if err == nil {
		t.Error("Expected error when key is not found")
	}
}

// TestLoadConfigurationInvalidYAML tests error handling with invalid YAML
func TestLoadConfigurationInvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `db:
  type: sqlite
  invalid yaml structure [[[
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := LoadConfiguration(configFile, DBKey)
	if err == nil {
		t.Error("Expected error when parsing invalid YAML")
	}
}

// TestLoadConfigurationUnmarshalError tests error handling with unmarshal errors
func TestLoadConfigurationUnmarshalError(t *testing.T) {
	// Create a config file with wrong data types
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `db:
  type: sqlite
  maxIdleConns: "not_a_number"
  maxOpenConns: "also_not_a_number"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err := LoadConfiguration(configFile, DBKey)
	if err == nil {
		t.Error("Expected error when unmarshaling invalid types")
	}
}

// TestCreateDBManagerSQLite tests creating a DB manager with SQLite
func TestCreateDBManagerSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	cfg := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile,
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 300,
	}

	manager, err := CreateDBManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create DB manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil DBManager")
	}

	if manager.Conn == nil {
		t.Fatal("Expected non-nil DB connection")
	}

	if manager.Config == nil {
		t.Fatal("Expected non-nil Config")
	}

	if manager.Config.Type != DBTypeSQLite {
		t.Errorf("Expected type %q, got %q", DBTypeSQLite, manager.Config.Type)
	}

	expectedDSN := dbFile
	if manager.DSN != expectedDSN {
		t.Errorf("Expected DSN %q, got %q", expectedDSN, manager.DSN)
	}
}

// TestCreateDBManagerInvalidConfig tests error handling with invalid database config
func TestCreateDBManagerInvalidConfig(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     DBTypePostgres,
		Host:     "nonexistent-host-12345.example.com",
		Port:     "5432",
		Name:     "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	_, err := CreateDBManager(cfg)
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

// TestCreateDBManagerFileSuccess tests creating DB manager from config file
func TestCreateDBManagerFileSuccess(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")
	dbFile := filepath.Join(tmpDir, "test.db")

	configContent := `db:
  type: sqlite
  filePath: ` + dbFile + `
  maxIdleConns: 3
  maxOpenConns: 8
  connMaxLifetime: 600
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	manager, err := CreateDBManagerFile(configFile)
	if err != nil {
		t.Fatalf("Failed to create DB manager from file: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected non-nil DBManager")
	}

	if manager.Conn == nil {
		t.Fatal("Expected non-nil DB connection")
	}

	if manager.Config.MaxIdleConns != 3 {
		t.Errorf("Expected MaxIdleConns 3, got %d", manager.Config.MaxIdleConns)
	}
}

// TestCreateDBManagerFileInvalidFile tests error handling with invalid file
func TestCreateDBManagerFileInvalidFile(t *testing.T) {
	_, err := CreateDBManagerFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent config file")
	}
}

// TestCheckSuccess tests successful database connection check
func TestCheckSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	cfg := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile,
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 300,
	}

	manager, err := CreateDBManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create DB manager: %v", err)
	}

	if err := manager.Check(); err != nil {
		t.Errorf("Expected successful check, got error: %v", err)
	}
}

// TestCheckFailure tests database check with closed connection
func TestCheckFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	cfg := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile,
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 300,
	}

	manager, err := CreateDBManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create DB manager: %v", err)
	}

	// Close the connection
	sqlDB, err := manager.Conn.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	// Check should now fail
	if err := manager.Check(); err == nil {
		t.Error("Expected error when checking closed connection")
	}
}

// TestGetDBSQLite tests GetDB method for SQLite
func TestGetDBSQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	cfg := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile,
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 300,
	}

	manager := &DBManager{
		Config: &cfg,
		DSN:    PrepareDSN(cfg),
	}

	dbConn, err := manager.GetDB()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if dbConn == nil {
		t.Fatal("Expected non-nil DB connection")
	}

	// Verify connection settings
	sqlDB, err := dbConn.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}

// TestGetDBMySQL tests GetDB method for MySQL (will fail to connect but tests the path)
func TestGetDBMySQL(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:            DBTypeMySQL,
		Host:            "invalid-mysql-host.example.com",
		Port:            "3306",
		Name:            "testdb",
		Username:        "testuser",
		Password:        "testpass",
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 300,
	}

	manager := &DBManager{
		Config: &cfg,
		DSN:    PrepareDSN(cfg),
	}

	// Should fail to connect to invalid host
	_, err := manager.GetDB()
	if err == nil {
		t.Error("Expected error when connecting to invalid MySQL host")
	}
}

// TestGetDBInvalidDriver tests GetDB with invalid database connection
func TestGetDBInvalidDriver(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     DBTypePostgres,
		Host:     "invalid-host-name-12345.example.com",
		Port:     "5432",
		Name:     "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	manager := &DBManager{
		Config: &cfg,
		DSN:    PrepareDSN(cfg),
	}

	_, err := manager.GetDB()
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

// TestGetDBWithDefaultType tests GetDB with empty/default database type
func TestGetDBWithDefaultType(t *testing.T) {
	cfg := config.ConfigurationDB{
		Type:     "", // Empty type should default to postgres
		Host:     "invalid-host.example.com",
		Port:     "5432",
		Name:     "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	manager := &DBManager{
		Config: &cfg,
		DSN:    PrepareDSN(cfg),
	}

	// Should attempt postgres connection (which will fail, but that's expected)
	_, err := manager.GetDB()
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

// TestDBManagerStructure tests the DBManager structure
func TestDBManagerStructure(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	cfg := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile,
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 300,
	}

	manager, err := CreateDBManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create DB manager: %v", err)
	}

	// Verify all fields are populated
	if manager.Conn == nil {
		t.Error("Expected non-nil Conn")
	}

	if manager.Config == nil {
		t.Error("Expected non-nil Config")
	}

	if manager.DSN == "" {
		t.Error("Expected non-empty DSN")
	}

	// Verify config values are preserved
	if manager.Config.Type != cfg.Type {
		t.Errorf("Expected Type %q, got %q", cfg.Type, manager.Config.Type)
	}

	if manager.Config.MaxIdleConns != cfg.MaxIdleConns {
		t.Errorf("Expected MaxIdleConns %d, got %d", cfg.MaxIdleConns, manager.Config.MaxIdleConns)
	}

	if manager.Config.MaxOpenConns != cfg.MaxOpenConns {
		t.Errorf("Expected MaxOpenConns %d, got %d", cfg.MaxOpenConns, manager.Config.MaxOpenConns)
	}

	if manager.Config.ConnMaxLifetime != cfg.ConnMaxLifetime {
		t.Errorf("Expected ConnMaxLifetime %d, got %d", cfg.ConnMaxLifetime, manager.Config.ConnMaxLifetime)
	}
}

// TestConnectionPoolSettings tests that connection pool settings are applied
func TestConnectionPoolSettings(t *testing.T) {
	tmpDir := t.TempDir()
	dbFile := filepath.Join(tmpDir, "test.db")

	cfg := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile,
		MaxIdleConns:    3,
		MaxOpenConns:    10,
		ConnMaxLifetime: 600,
	}

	manager, err := CreateDBManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create DB manager: %v", err)
	}

	sqlDB, err := manager.Conn.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	// Verify the settings are applied (these will be reflected in the stats)
	stats := sqlDB.Stats()

	// MaxOpenConnections should be set
	if stats.MaxOpenConnections != 10 {
		t.Errorf("Expected MaxOpenConnections 10, got %d", stats.MaxOpenConnections)
	}
}

// TestMultipleDBManagers tests creating multiple independent DB managers
func TestMultipleDBManagers(t *testing.T) {
	tmpDir := t.TempDir()

	dbFile1 := filepath.Join(tmpDir, "test1.db")
	cfg1 := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile1,
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 300,
	}

	dbFile2 := filepath.Join(tmpDir, "test2.db")
	cfg2 := config.ConfigurationDB{
		Type:            DBTypeSQLite,
		FilePath:        dbFile2,
		MaxIdleConns:    3,
		MaxOpenConns:    8,
		ConnMaxLifetime: 600,
	}

	manager1, err := CreateDBManager(cfg1)
	if err != nil {
		t.Fatalf("Failed to create first DB manager: %v", err)
	}

	manager2, err := CreateDBManager(cfg2)
	if err != nil {
		t.Fatalf("Failed to create second DB manager: %v", err)
	}

	// Verify they're independent
	if manager1.DSN == manager2.DSN {
		t.Error("Expected different DSNs for different managers")
	}

	if manager1.Config.MaxIdleConns == manager2.Config.MaxIdleConns {
		t.Error("Expected different MaxIdleConns for different managers")
	}

	// Both should be functional
	if err := manager1.Check(); err != nil {
		t.Errorf("Manager1 check failed: %v", err)
	}

	if err := manager2.Check(); err != nil {
		t.Errorf("Manager2 check failed: %v", err)
	}
}

// TestConstants tests that all constants are properly defined
func TestConstants(t *testing.T) {
	if PostgresDBString == "" {
		t.Error("PostgresDBString should not be empty")
	}

	if MySQLDBString == "" {
		t.Error("MySQLDBString should not be empty")
	}

	if DBKey == "" {
		t.Error("DBKey should not be empty")
	}

	if DBTypePostgres == "" {
		t.Error("DBTypePostgres should not be empty")
	}

	if DBTypeMySQL == "" {
		t.Error("DBTypeMySQL should not be empty")
	}

	if DBTypeSQLite == "" {
		t.Error("DBTypeSQLite should not be empty")
	}

	// Verify expected constant values
	if DBKey != "db" {
		t.Errorf("Expected DBKey to be 'db', got %q", DBKey)
	}

	if DBTypePostgres != "postgres" {
		t.Errorf("Expected DBTypePostgres to be 'postgres', got %q", DBTypePostgres)
	}

	if DBTypeMySQL != "mysql" {
		t.Errorf("Expected DBTypeMySQL to be 'mysql', got %q", DBTypeMySQL)
	}

	if DBTypeSQLite != "sqlite" {
		t.Errorf("Expected DBTypeSQLite to be 'sqlite', got %q", DBTypeSQLite)
	}
}
