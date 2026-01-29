package config

import (
	"fmt"
	"strconv"
)

var validAuthMechanisms = map[string]struct{}{
	AuthNone: {},
	AuthDB:   {},
}

var validDBTypes = map[string]struct{}{
	DBTypePostgres: {},
	DBTypeMySQL:    {},
	DBTypeSQLite:   {},
}

var validLogLevels = map[string]struct{}{
	LogLevelDebug: {},
	LogLevelInfo:  {},
	LogLevelWarn:  {},
	LogLevelError: {},
}

var validLogFormats = map[string]struct{}{
	LogFormatConsole: {},
	LogFormatJSON:    {},
}

// Validate that required pieces of the configuration are set
func ValidateConfigValues(cfg MapCTFConfiguration) error {
	if cfg.Service.Listener == "" {
		return fmt.Errorf("service.listener cannot be empty")
	}
	if cfg.Service.Port == "" {
		return fmt.Errorf("service.port cannot be empty")
	}
	if _, err := strconv.Atoi(cfg.Service.Port); err != nil {
		return fmt.Errorf("service.port must be numeric: %w", err)
	}
	if _, ok := validLogLevels[cfg.Service.LogLevel]; !ok {
		return fmt.Errorf("service.logLevel %q is not supported", cfg.Service.LogLevel)
	}
	if _, ok := validLogFormats[cfg.Service.LogFormat]; !ok {
		return fmt.Errorf("service.logFormat %q is not supported", cfg.Service.LogFormat)
	}
	if _, ok := validAuthMechanisms[cfg.Service.Auth]; !ok {
		return fmt.Errorf("service.auth %q is not supported", cfg.Service.Auth)
	}
	if err := ValidateDBConfig(cfg.DB); err != nil {
		return err
	}
	if err := ValidateRedisConfig(cfg.Redis); err != nil {
		return err
	}
	if cfg.Metrics.Enabled {
		if cfg.Metrics.Listener == "" {
			return fmt.Errorf("metrics.listener cannot be empty when metrics are enabled")
		}
		if cfg.Metrics.Port == "" {
			return fmt.Errorf("metrics.port cannot be empty when metrics are enabled")
		}
		if _, err := strconv.Atoi(cfg.Metrics.Port); err != nil {
			return fmt.Errorf("metrics.port must be numeric: %w", err)
		}
	}
	if cfg.TLS.Termination {
		if cfg.TLS.CertificateFile == "" || cfg.TLS.KeyFile == "" {
			return fmt.Errorf("tls.certificateFile and tls.keyFile are required when TLS termination is enabled")
		}
	}
	if cfg.DebugHTTP.Enabled {
		if cfg.DebugHTTP.File == "" {
			return fmt.Errorf("debugHttp.file cannot be empty when HTTP debug logging is enabled")
		}
	}
	return nil
}

func ValidateDBConfig(dbCfg ConfigurationDB) error {
	if _, ok := validDBTypes[dbCfg.Type]; !ok {
		return fmt.Errorf("db.type %q is not supported", dbCfg.Type)
	}
	switch dbCfg.Type {
	case DBTypeSQLite:
		if dbCfg.FilePath == "" {
			return fmt.Errorf("db.filePath must be set when db.type is sqlite")
		}
	case DBTypePostgres, DBTypeMySQL:
		if dbCfg.Host == "" {
			return fmt.Errorf("db.host cannot be empty")
		}
		if dbCfg.Port == "" {
			return fmt.Errorf("db.port cannot be empty")
		}
		if _, err := strconv.Atoi(dbCfg.Port); err != nil {
			return fmt.Errorf("db.port must be numeric: %w", err)
		}
		if dbCfg.Name == "" {
			return fmt.Errorf("db.name cannot be empty")
		}
		if dbCfg.Username == "" {
			return fmt.Errorf("db.username cannot be empty")
		}
		if dbCfg.Password == "" {
			return fmt.Errorf("db.password cannot be empty")
		}
	default:
		return fmt.Errorf("db.type %q is not supported", dbCfg.Type)
	}
	return nil
}

func ValidateRedisConfig(redisCfg ConfigurationRedis) error {
	if redisCfg.ConnectionString != "" {
		return nil
	}
	if redisCfg.Host == "" {
		return fmt.Errorf("redis.host cannot be empty when connectionString is not provided")
	}
	if redisCfg.Port == "" {
		return fmt.Errorf("redis.port cannot be empty when connectionString is not provided")
	}
	if _, err := strconv.Atoi(redisCfg.Port); err != nil {
		return fmt.Errorf("redis.port must be numeric: %w", err)
	}
	return nil
}
