package config

import (
	"github.com/urfave/cli/v3"
)

// Default values
const (
	// Default timeout to attempt backend reconnect
	defaultBackendRetryTimeout int = 10
	// Default timeout to attempt redis reconnect
	defaultRedisRetryTimeout int = 10
)

// osquery
const (
	// Default service configuration file
	defServiceConfigurationFile string = "config/mapctf.yaml"
	// Default TLS certificate file
	defTLSCertificateFile string = "config/certs/tls.crt"
	// Default TLS private key file
	defTLSKeyFile string = "config/certs/tls.key"
	// Static files folder
	defStaticFilesFolder string = "./static"
	// Default templates folder
	defTemplatesFolder string = "./templates"
	// Default db filepath for sqlite
	defSQLiteDBFile string = "./mapctf.db"
	// Default debug HTTP file
	defDebugHTTPFile string = "debug-http-mapctf.log"
)

// ServiceFlagParams stores flag values for the each service
type ServiceFlagParams struct {
	// Configuration will be loaded from a file
	ConfigFileFlag bool
	// Service configuration file
	ConfigFile string
	// Service configuration values from YAML
	ConfigValues MapCTFConfiguration
}

// InitMapFlags initializes all the flags needed for the TLS service
func InitMapFlags(params *ServiceFlagParams) []cli.Flag {
	var allFlags []cli.Flag
	// Add flags by category
	allFlags = append(allFlags, initConfigFlags(params)...)
	allFlags = append(allFlags, initServiceFlags(params)...)
	allFlags = append(allFlags, initMetricsFlags(params)...)
	allFlags = append(allFlags, initRedisFlags(params)...)
	allFlags = append(allFlags, initDBFlags(params)...)
	allFlags = append(allFlags, initTLSSecurityFlags(params)...)
	allFlags = append(allFlags, initJWTFlags(params)...)
	allFlags = append(allFlags, initDebugFlags(params)...)
	return allFlags
}

// initConfigFlags initializes configuration-related flags
func initConfigFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Value:       false,
			Usage:       "Provide service configuration via YAML file",
			Sources:     cli.EnvVars("SERVICE_CONFIG"),
			Destination: &params.ConfigFileFlag,
		},
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"C"},
			Value:       defServiceConfigurationFile,
			Usage:       "Load YAML service configuration from `FILE`",
			Sources:     cli.EnvVars("SERVICE_CONFIG_FILE"),
			Destination: &params.ConfigFile,
		},
	}
}

// initServiceFlags initializes main service-related flags
func initServiceFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "listener",
			Aliases:     []string{"l"},
			Value:       "0.0.0.0",
			Usage:       "Listener for the service",
			Sources:     cli.EnvVars("SERVICE_LISTENER"),
			Destination: &params.ConfigValues.Service.Listener,
		},
		&cli.StringFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			Value:       "9000",
			Usage:       "TCP port for the service",
			Sources:     cli.EnvVars("SERVICE_PORT"),
			Destination: &params.ConfigValues.Service.Port,
		},
		&cli.StringFlag{
			Name:        "host",
			Aliases:     []string{"H"},
			Value:       "0.0.0.0",
			Usage:       "Exposed hostname the service uses",
			Sources:     cli.EnvVars("SERVICE_HOST"),
			Destination: &params.ConfigValues.Service.Host,
		},
		&cli.StringFlag{
			Name:        "auth",
			Aliases:     []string{"A"},
			Value:       AuthNone,
			Usage:       "Authentication mechanism for the service",
			Sources:     cli.EnvVars("SERVICE_AUTH"),
			Destination: &params.ConfigValues.Service.Auth,
		},
		&cli.StringFlag{
			Name:        "log-level",
			Value:       LogLevelInfo,
			Usage:       "Log level for the service",
			Sources:     cli.EnvVars("SERVICE_LOG_LEVEL"),
			Destination: &params.ConfigValues.Service.LogLevel,
		},
		&cli.StringFlag{
			Name:        "log-format",
			Value:       LogFormatConsole,
			Usage:       "Log format for the service",
			Sources:     cli.EnvVars("SERVICE_LOG_FORMAT"),
			Destination: &params.ConfigValues.Service.LogFormat,
		},
		&cli.StringFlag{
			Name:        "uuid",
			Value:       "",
			Usage:       "UUID to be used for the service",
			Sources:     cli.EnvVars("SERVICE_UUID"),
			Destination: &params.ConfigValues.Service.UUID,
		},
	}
}

// initMetricsFlags initializes metrics-related flags
func initMetricsFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "metrics-listener",
			Value:       "0.0.0.0",
			Usage:       "Listener for prometheus metrics",
			Sources:     cli.EnvVars("METRICS_LISTENER"),
			Destination: &params.ConfigValues.Metrics.Listener,
		},
		&cli.StringFlag{
			Name:        "metrics-port",
			Value:       "9090",
			Usage:       "Port for exposing prometheus metrics",
			Sources:     cli.EnvVars("METRICS_PORT"),
			Destination: &params.ConfigValues.Metrics.Port,
		},
		&cli.BoolFlag{
			Name:        "metrics-enabled",
			Value:       false,
			Usage:       "Enable prometheus metrics",
			Sources:     cli.EnvVars("METRICS_ENABLED"),
			Destination: &params.ConfigValues.Metrics.Enabled,
		},
	}
}

// initRedisFlags initializes Redis-related flags
func initRedisFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "redis-connection-string",
			Value:       "",
			Usage:       "Redis connection string, must include schema (<redis|rediss|unix>://<user>:<pass>@<host>:<port>/<db>?<options>",
			Sources:     cli.EnvVars("REDIS_CONNECTION_STRING"),
			Destination: &params.ConfigValues.Redis.ConnectionString,
		},
		&cli.StringFlag{
			Name:        "redis-host",
			Value:       "127.0.0.1",
			Usage:       "Redis host to be connected to",
			Sources:     cli.EnvVars("REDIS_HOST"),
			Destination: &params.ConfigValues.Redis.Host,
		},
		&cli.StringFlag{
			Name:        "redis-port",
			Value:       "6379",
			Usage:       "Redis port to be connected to",
			Sources:     cli.EnvVars("REDIS_PORT"),
			Destination: &params.ConfigValues.Redis.Port,
		},
		&cli.StringFlag{
			Name:        "redis-pass",
			Value:       "",
			Usage:       "Password to be used for redis",
			Sources:     cli.EnvVars("REDIS_PASS"),
			Destination: &params.ConfigValues.Redis.Password,
		},
		&cli.IntFlag{
			Name:        "redis-db",
			Value:       0,
			Usage:       "Redis database to be selected after connecting",
			Sources:     cli.EnvVars("REDIS_DB"),
			Destination: &params.ConfigValues.Redis.DB,
		},
		&cli.IntFlag{
			Name:        "redis-conn-retry",
			Value:       defaultRedisRetryTimeout,
			Usage:       "Time in seconds to retry the connection to the cache, if set to 0 the service will stop if the connection fails",
			Sources:     cli.EnvVars("REDIS_CONN_RETRY"),
			Destination: &params.ConfigValues.Redis.ConnRetry,
		},
	}
}

// initDBFlags initializes database-related flags
func initDBFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "db-type",
			Value:       "postgres",
			Usage:       "Type of backend to be used",
			Sources:     cli.EnvVars("DB_TYPE"),
			Destination: &params.ConfigValues.DB.Type,
		},
		&cli.StringFlag{
			Name:        "db-host",
			Value:       "127.0.0.1",
			Usage:       "Backend host to be connected to",
			Sources:     cli.EnvVars("DB_HOST"),
			Destination: &params.ConfigValues.DB.Host,
		},
		&cli.StringFlag{
			Name:        "db-port",
			Value:       "5432",
			Usage:       "Backend port to be connected to",
			Sources:     cli.EnvVars("DB_PORT"),
			Destination: &params.ConfigValues.DB.Port,
		},
		&cli.StringFlag{
			Name:        "db-name",
			Value:       "mapctf",
			Usage:       "Database name to be used in the backend",
			Sources:     cli.EnvVars("DB_NAME"),
			Destination: &params.ConfigValues.DB.Name,
		},
		&cli.StringFlag{
			Name:        "db-user",
			Value:       "mapctf-user",
			Usage:       "Username to be used for the backend",
			Sources:     cli.EnvVars("DB_USER"),
			Destination: &params.ConfigValues.DB.Username,
		},
		&cli.StringFlag{
			Name:        "db-pass",
			Value:       "mapctf-pass",
			Usage:       "Password to be used for the backend",
			Sources:     cli.EnvVars("DB_PASS"),
			Destination: &params.ConfigValues.DB.Password,
		},
		&cli.StringFlag{
			Name:        "db-sslmode",
			Value:       "disable",
			Usage:       "SSL native support to encrypt the connection to the backend",
			Sources:     cli.EnvVars("DB_SSLMODE"),
			Destination: &params.ConfigValues.DB.SSLMode,
		},
		&cli.IntFlag{
			Name:        "db-max-idle-conns",
			Value:       20,
			Usage:       "Maximum number of connections in the idle connection pool",
			Sources:     cli.EnvVars("DB_MAX_IDLE_CONNS"),
			Destination: &params.ConfigValues.DB.MaxIdleConns,
		},
		&cli.IntFlag{
			Name:        "db-max-open-conns",
			Value:       100,
			Usage:       "Maximum number of open connections to the database",
			Sources:     cli.EnvVars("DB_MAX_OPEN_CONNS"),
			Destination: &params.ConfigValues.DB.MaxOpenConns,
		},
		&cli.IntFlag{
			Name:        "db-conn-max-lifetime",
			Value:       30,
			Usage:       "Maximum amount of time a connection may be reused",
			Sources:     cli.EnvVars("DB_CONN_MAX_LIFETIME"),
			Destination: &params.ConfigValues.DB.ConnMaxLifetime,
		},
		&cli.IntFlag{
			Name:        "db-conn-retry",
			Value:       defaultBackendRetryTimeout,
			Usage:       "Time in seconds to retry the connection to the database, if set to 0 the service will stop if the connection fails",
			Sources:     cli.EnvVars("DB_CONN_RETRY"),
			Destination: &params.ConfigValues.DB.ConnRetry,
		},
		&cli.StringFlag{
			Name:        "db-filepath",
			Value:       defSQLiteDBFile,
			Usage:       "File path to the SQLite database, only used when type is sqlite",
			Sources:     cli.EnvVars("DB_SQLITE_FILEPATH"),
			Destination: &params.ConfigValues.DB.FilePath,
		},
	}
}

// initTLSSecurityFlags initializes TLS security-related flags
func initTLSSecurityFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "tls",
			Aliases:     []string{"t"},
			Value:       false,
			Usage:       "Enable TLS termination. It requires certificate and key",
			Sources:     cli.EnvVars("TLS_SERVER"),
			Destination: &params.ConfigValues.TLS.Termination,
		},
		&cli.StringFlag{
			Name:        "cert",
			Aliases:     []string{"T"},
			Value:       defTLSCertificateFile,
			Usage:       "TLS termination certificate from `FILE`",
			Sources:     cli.EnvVars("TLS_CERTIFICATE"),
			Destination: &params.ConfigValues.TLS.CertificateFile,
		},
		&cli.StringFlag{
			Name:        "key",
			Aliases:     []string{"K"},
			Value:       defTLSKeyFile,
			Usage:       "TLS termination private key from `FILE`",
			Sources:     cli.EnvVars("TLS_KEY"),
			Destination: &params.ConfigValues.TLS.KeyFile,
		},
	}
}

// initJWTFlags initializes JWT-related flags
func initJWTFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "jwt-secret",
			Value:       "",
			Usage:       "Secret key to be used for JWT",
			Sources:     cli.EnvVars("JWT_SECRET"),
			Destination: &params.ConfigValues.JWT.Secret,
		},
		&cli.IntFlag{
			Name:        "jwt-expiration",
			Value:       3600,
			Usage:       "Expiration time in seconds for JWT",
			Sources:     cli.EnvVars("JWT_EXPIRATION"),
			Destination: &params.ConfigValues.JWT.HoursToExpire,
		},
	}
}

// initMapFlags initializes map-related flags
func initMapFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "session-key",
			Value:       "",
			Usage:       "Session key to generate cookies from it",
			Sources:     cli.EnvVars("SESSION_KEY"),
			Destination: &params.ConfigValues.Map.SessionKey,
		},
		&cli.StringFlag{
			Name:        "static",
			Aliases:     []string{"s"},
			Value:       defStaticFilesFolder,
			Usage:       "Directory with all the static files needed for the osctrl-admin UI",
			Sources:     cli.EnvVars("STATIC_FILES"),
			Destination: &params.ConfigValues.Map.StaticDir,
		},
		&cli.StringFlag{
			Name:        "templates",
			Value:       defTemplatesFolder,
			Usage:       "Directory with all the templates needed for the osctrl-admin UI",
			Sources:     cli.EnvVars("TEMPLATES_DIR"),
			Destination: &params.ConfigValues.Map.TemplatesDir,
		},
	}
}

// initDebugFlags initializes all the debug logging specific flags
func initDebugFlags(params *ServiceFlagParams) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "enable-http-debug",
			Value:       false,
			Usage:       "Enable HTTP Debug mode to dump full HTTP incoming request",
			Sources:     cli.EnvVars("HTTP_DEBUG"),
			Destination: &params.ConfigValues.DebugHTTP.Enabled,
		},
		&cli.StringFlag{
			Name:        "http-debug-file",
			Value:       defDebugHTTPFile,
			Usage:       "File to dump the HTTP requests when HTTP Debug mode is enabled",
			Sources:     cli.EnvVars("HTTP_DEBUG_FILE"),
			Destination: &params.ConfigValues.DebugHTTP.File,
		},
		&cli.BoolFlag{
			Name:        "http-debug-show-body",
			Value:       false,
			Usage:       "Show body of the HTTP requests when HTTP Debug mode is enabled",
			Sources:     cli.EnvVars("HTTP_DEBUG_SHOW_BODY"),
			Destination: &params.ConfigValues.DebugHTTP.ShowBody,
		},
	}
}
