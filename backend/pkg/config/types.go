package config

// Types of logging
const (
	// log levels
	LogLevelDebug string = "debug"
	LogLevelInfo  string = "info"
	LogLevelWarn  string = "warn"
	LogLevelError string = "error"
	// log formats
	LogFormatConsole string = "console"
	LogFormatJSON    string = "json"
)

// Types of authentication
const (
	AuthNone string = "none"
	AuthDB   string = "db"
)

// Types of backend
const (
	DBTypePostgres string = "postgres"
	DBTypeMySQL    string = "mysql"
	DBTypeSQLite   string = "sqlite"
)

// MapCTFConfiguration to hold all configuration values
type MapCTFConfiguration struct {
	ServiceConfigFile string                 `yaml:"-"`
	Service           ConfigurationService   `mapstructure:"service"`
	DB                ConfigurationDB        `mapstructure:"db"`
	Redis             ConfigurationRedis     `mapstructure:"redis"`
	Metrics           ConfigurationMetrics   `mapstructure:"metrics"`
	TLS               ConfigurationTLS       `mapstructure:"tls"`
	JWT               ConfigurationJWT       `mapstructure:"jwt"`
	Map               ConfigurationMap       `mapstructure:"map"`
	DebugHTTP         ConfigurationDebugHTTP `mapstructure:"debugHttp"`
}

// ConfigurationService to hold the service configuration values
type ConfigurationService struct {
	Listener  string `yaml:"listener"`
	Port      string `yaml:"port"`
	LogLevel  string `yaml:"logLevel"`
	LogFormat string `yaml:"logFormat"`
	Host      string `yaml:"host"`
	Auth      string `yaml:"auth"`
	UUID      string `yaml:"uuid"`
}

// ConfigurationDB to hold all backend configuration values
type ConfigurationDB struct {
	Type            string `yaml:"type"` // Database type: postgres, mysql, sqlite
	Host            string `yaml:"host"`
	Port            string `yaml:"port"`
	Name            string `yaml:"name"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	SSLMode         string `yaml:"sslmode"` // For postgres
	MaxIdleConns    int    `yaml:"maxIdleConns"`
	MaxOpenConns    int    `yaml:"maxOpenConns"`
	ConnMaxLifetime int    `yaml:"connMaxLifetime"`
	ConnRetry       int    `yaml:"connRetry"`
	FilePath        string `yaml:"filePath"` // Used for SQLite
}

// ConfigurationRedis to hold all redis configuration values
type ConfigurationRedis struct {
	Host             string `yaml:"host"`
	Port             string `yaml:"port"`
	Password         string `yaml:"password"`
	ConnectionString string `yaml:"connectionString"`
	DB               int    `yaml:"db"`
	ConnRetry        int    `yaml:"connRetry"`
}

// ConfigurationMetrics to hold the metrics configuration values
type ConfigurationMetrics struct {
	Enabled  bool   `yaml:"enabled"`
	Listener string `yaml:"listener"`
	Port     string `yaml:"port"`
}

// ConfigurationTLS to hold the TLS/SSL termination configuration values
type ConfigurationTLS struct {
	Termination     bool   `yaml:"termination"`
	CertificateFile string `yaml:"certificateFile"`
	KeyFile         string `yaml:"keyFile"`
}

// ConfigurationJWT to hold the JWT configuration values
type ConfigurationJWT struct {
	Secret        string `yaml:"secret"`
	HoursToExpire int    `yaml:"hoursToExpire"`
}

// ConfigurationMap to hold the map configuration values
type ConfigurationMap struct {
	SessionKey   string `yaml:"sessionKey"`
	StaticDir    string `yaml:"staticDir"`
	TemplatesDir string `yaml:"templatesDir"`
}

// ConfigurationDebugHTTP to hold all HTTP debug configuration values
type ConfigurationDebugHTTP struct {
	Enabled  bool   `yaml:"enabled"`
	File     string `yaml:"file"`
	ShowBody bool   `yaml:"showBody"`
}
