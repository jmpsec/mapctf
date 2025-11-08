package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jmpsec/mapctf/cmd/api/handlers"
	"github.com/jmpsec/mapctf/pkg/backend"
	"github.com/jmpsec/mapctf/pkg/cache"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

const (
	// Project name
	projectName = "mapCTF"
	// Service name
	serviceName = projectName + "-api"
	// Service version
	serviceVersion = version.MapCTFVersion
	// Service description
	serviceDescription = "API service for " + projectName
	// Application description
	appDescription = serviceDescription + " a map-based CTF platform"
	// Default time format for loggers
	LoggerTimeFormat string = "2006-01-02T15:04:05.999Z07:00"
	// Default path used when generating example configs
	defaultExampleConfigPath = "config/mapctf.example.yaml"
)

var validAuthMechanisms = map[string]struct{}{
	config.AuthNone:  {},
	config.AuthDB:    {},
	config.AuthSAML:  {},
	config.AuthJWT:   {},
	config.AuthOAuth: {},
	config.AuthOIDC:  {},
}

// Paths
const (
	// Default endpoint to handle root
	rootPath string = "/"
	// Default endpoint to handle HTTP health
	healthPath string = "/health"
	// Default endpoint to handle Login
	loginPath string = "/auth/login"
	// Default endpoint to handle Logout
	logoutPath string = "/auth/logout"
	// Default endpoint to handle HTTP(500) errors
	errorPath string = "/error"
	// Default endpoint to handle Forbidden(403) errors
	forbiddenPath string = "/forbidden"
	// API checks path
	checksNoAuthPath = "/checks-no-auth"
	checksAuthPath   = "/checks-auth"
	// API prefix path
	apiPrefixPath = "/api"
	// API version path
	apiVersionPath = "/v1"
	// API admin path
	apiAdminPath = "/admin"
	// API stats path
	apiStatsPath = "/stats"
	// API teams path
	apiTeamsPath = "/teams"
	// API challenges path
	apiChallengesPath = "/challenges"
)

// Build-time metadata (overridden via -ldflags "-X main.buildVersion=... -X main.buildCommit=... -X main.buildDate=...")
var (
	buildVersion = version.MapCTFVersion
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

// Global general variables
var (
	err        error
	db         *backend.DBManager
	redis      *cache.RedisManager
	app        *cli.Command
	flags      []cli.Flag
	flagParams config.ServiceFlagParams
)

// Function to load the configuration from a single YAML file
func loadConfigurationYAML(file string) (config.MapCTFConfiguration, error) {
	var cfg config.MapCTFConfiguration
	// Load file and read config
	viper.SetConfigFile(file)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		return cfg, err
	}
	// Unmarshal into struct
	if err := viper.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	// No errors!
	return cfg, nil
}

// Validate that required pieces of the configuration are set
func validateConfigValues(cfg config.MapCTFConfiguration) error {
	if cfg.Service.Listener == "" {
		return fmt.Errorf("service.listener cannot be empty")
	}
	if cfg.Service.Port == "" {
		return fmt.Errorf("service.port cannot be empty")
	}
	if _, err := strconv.Atoi(cfg.Service.Port); err != nil {
		return fmt.Errorf("service.port must be numeric: %w", err)
	}
	if _, ok := validAuthMechanisms[cfg.Service.Auth]; !ok {
		return fmt.Errorf("service.auth %q is not supported", cfg.Service.Auth)
	}
	if err := validateDBConfig(cfg.DB); err != nil {
		return err
	}
	if err := validateRedisConfig(cfg.Redis); err != nil {
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

func validateDBConfig(dbCfg config.ConfigurationDB) error {
	switch dbCfg.Type {
	case config.DBTypeSQLite:
		if dbCfg.FilePath == "" {
			return fmt.Errorf("db.filePath must be set when db.type is sqlite")
		}
	case config.DBTypePostgres, config.DBTypeMySQL:
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

func validateRedisConfig(redisCfg config.ConfigurationRedis) error {
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

func configFileFromCommand(cmd *cli.Command) string {
	if file := cmd.String("file"); file != "" {
		return file
	}
	if flagParams.ConfigFile != "" {
		return flagParams.ConfigFile
	}
	return "config/mapctf.yaml"
}

func generateExampleConfigFile(path string, cfg config.MapCTFConfiguration, overwrite bool) error {
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

// Initialization code
func init() {
	// Initialize CLI flags using the config package
	flags = config.InitMapFlags(&flagParams)
}

/*
POST   /api/auth/login          - User login
GET    /api/logout              - User logout
GET    /api/teams               - List teams
GET    /api/challenges          - List challenges
GET    /api/admin/stats         - Admin stats
GET    /api/admin/teams         - Manage teams
POST   /api/admin/teams         - Create team
DELETE /api/admin/teams/:id     - Delete team
GET    /api/admin/challenges    - Manage challenges
POST   /api/admin/challenges    - Create challenge
PATCH  /api/admin/challenges/:id - Update challenge
DELETE /api/admin/challenges/:id - Delete challenge
*/

// Let's go!
func mapCTFService() {
	// ////////////////////////////// Backend
	log.Info().Msg("Initializing backend...")
	for {
		db, err = backend.CreateDBManager(flagParams.ConfigValues.DB)
		if db != nil {
			log.Info().Msg("Connection to backend successful!")
			break
		}
		if err != nil {
			log.Err(err).Msg("Failed to connect to backend")
			if flagParams.ConfigValues.DB.ConnRetry == 0 {
				log.Fatal().Msg("Connection to backend failed and no retry was set")
			}
		}
		log.Debug().Msgf("Backend NOT ready! Retrying in %d seconds...\n", flagParams.ConfigValues.DB.ConnRetry)
		time.Sleep(time.Duration(flagParams.ConfigValues.DB.ConnRetry) * time.Second)
	}
	// ////////////////////////////// Cache
	log.Info().Msg("Initializing cache...")
	for {
		redis, err = cache.CreateRedisManager(flagParams.ConfigValues.Redis)
		if redis != nil {
			log.Info().Msg("Connection to cache successful!")
			break
		}
		if err != nil {
			log.Err(err).Msg("Failed to connect to cache")
			if flagParams.ConfigValues.Redis.ConnRetry == 0 {
				log.Fatal().Msg("Connection to cache failed and no retry was set")
			}
		}
		log.Debug().Msgf("Cache NOT ready! Retrying in %d seconds...\n", flagParams.ConfigValues.Redis.ConnRetry)
		time.Sleep(time.Duration(flagParams.ConfigValues.Redis.ConnRetry) * time.Second)
	}
	// ////////////////////////////// Handlers
	log.Info().Msg("Initializing handlers")
	handlersCTF := handlers.CreateHandlersAPI(
		handlers.WithDB(db.Conn),
		handlers.WithRedisCache(redis),
		handlers.WithConfig(flagParams.ConfigValues),
		handlers.WithDebugHTTP(&flagParams.ConfigValues.DebugHTTP),
	)
	// ////////////////////////////// Router
	log.Info().Msg("Initializing router")
	// Create router for API
	muxAPI := http.NewServeMux()
	// Root
	muxAPI.HandleFunc("GET /", handlersCTF.RootHandler)
	// Testing
	muxAPI.HandleFunc("GET "+healthPath, handlersCTF.HealthHandler)
	// Rrror
	muxAPI.HandleFunc("GET "+errorPath, handlersCTF.ErrorHandler)
	// Forbidden
	muxAPI.HandleFunc("GET "+forbiddenPath, handlersCTF.ForbiddenHandler)
	// Check status
	muxAPI.HandleFunc("GET "+_apiPath(checksNoAuthPath), handlersCTF.CheckHandlerNoAuth)
	// Login
	muxAPI.HandleFunc("POST "+_apiPath(loginPath), handlersCTF.LoginHandler)
	// Logout
	muxAPI.HandleFunc("GET "+_apiPath(logoutPath), handlersCTF.LogoutHandler)
	// Teams
	muxAPI.HandleFunc("GET "+_apiPath(apiTeamsPath), handlersCTF.GetTeamsHandler)
	// Challenges
	muxAPI.HandleFunc("GET "+_apiPath(apiChallengesPath), handlersCTF.GetChallengesHandler)
	// Admin Stats
	muxAPI.HandleFunc("GET "+_apiPath(apiAdminPath+apiStatsPath), handlersCTF.GetAdminStatsHandler)
	// Admin Teams
	muxAPI.HandleFunc("GET "+_apiPath(apiAdminPath+apiTeamsPath), handlersCTF.GetAdminTeamsHandler)
	muxAPI.HandleFunc("POST "+_apiPath(apiAdminPath+apiTeamsPath), handlersCTF.CreateAdminTeamHandler)
	muxAPI.HandleFunc("DELETE "+_apiPath(apiAdminPath+apiTeamsPath), handlersCTF.DeleteAdminTeamHandler)
	// Admin Challenges
	muxAPI.HandleFunc("GET "+_apiPath(apiAdminPath+apiChallengesPath), handlersCTF.GetAdminChallengesHandler)
	muxAPI.HandleFunc("POST "+_apiPath(apiAdminPath+apiChallengesPath), handlersCTF.CreateAdminChallengeHandler)
	muxAPI.HandleFunc("PATCH "+_apiPath(apiAdminPath+apiChallengesPath), handlersCTF.UpdateAdminChallengeHandler)
	muxAPI.HandleFunc("DELETE "+_apiPath(apiAdminPath+apiChallengesPath), handlersCTF.DeleteAdminChallengeHandler)
	// Launch HTTP server for admin
	serviceListener := flagParams.ConfigValues.Service.Listener + ":" + flagParams.ConfigValues.Service.Port
	if flagParams.ConfigValues.TLS.Termination {
		cfg := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		srv := &http.Server{
			Addr:         serviceListener,
			Handler:      muxAPI,
			TLSConfig:    cfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
		log.Info().Msgf("%s v%s - HTTPS listening %s", serviceName, buildVersion, serviceListener)
		log.Info().Msgf("%s - commit=%s - build date=%s", serviceName, buildCommit, buildDate)
		if err := srv.ListenAndServeTLS(flagParams.ConfigValues.TLS.CertificateFile, flagParams.ConfigValues.TLS.KeyFile); err != nil {
			log.Fatal().Msgf("ListenAndServeTLS: %v", err)
		}
	} else {
		log.Info().Msgf("%s v%s - HTTP listening %s", serviceName, buildVersion, serviceListener)
		log.Info().Msgf("%s - commit=%s - build date=%s", serviceName, buildCommit, buildDate)
		if err := http.ListenAndServe(serviceListener, muxAPI); err != nil {
			log.Fatal().Msgf("ListenAndServe: %v", err)
		}
	}
}

// Action to run when no flags are provided to run checks and prepare data
func cliAction() error {
	// Load configuration if external config file is used
	if flagParams.ConfigFileFlag {
		flagParams.ConfigValues, err = loadConfigurationYAML(flagParams.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load service configuration %s - %w", flagParams.ConfigFile, err)
		}
	}
	if err := validateConfigValues(flagParams.ConfigValues); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	return nil
}

func initializeLoggers(cfg config.MapCTFConfiguration) {
	// Set the log level
	switch strings.ToLower(cfg.Service.LogLevel) {
	case config.LogLevelDebug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case config.LogLevelInfo:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case config.LogLevelWarn:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case config.LogLevelError:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	// Set the log format
	switch strings.ToLower(cfg.Service.LogFormat) {
	case config.LogFormatJSON:
		log.Logger = log.With().Caller().Logger()
	case config.LogFormatConsole:
		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			return filepath.Base(file) + ":" + strconv.Itoa(line)
		}
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: LoggerTimeFormat}).With().Caller().Logger()
	default:
		log.Logger = log.With().Caller().Logger()
	}
}

func main() {
	// Initiate CLI and parse arguments
	app = &cli.Command{
		Name:                  serviceName,
		Usage:                 appDescription,
		Version:               buildVersion,
		Description:           appDescription,
		Flags:                 flags,
		EnableShellCompletion: true,
	}
	// Customize version output (supports `--version` and `version` command)
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("%s version=%s commit=%s date=%s\n", serviceName, buildVersion, buildCommit, buildDate)
	}
	// Add -v alias to the global --version flag
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Print version information",
	}
	// Define this command for help to exit when help flag is passed
	app.Commands = []*cli.Command{
		{
			Name: "help",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				cli.ShowAppHelpAndExit(cmd, 0)
				return nil
			},
		},
		{
			Name:  "config-validate",
			Usage: "Validate a MapCTF configuration file",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "file",
					Aliases: []string{"f"},
					Usage:   "Path to the configuration file to validate",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				path := configFileFromCommand(cmd)
				cfg, err := loadConfigurationYAML(path)
				if err != nil {
					return fmt.Errorf("failed to load %s: %w", path, err)
				}
				if err := validateConfigValues(cfg); err != nil {
					return fmt.Errorf("configuration %s is invalid: %w", path, err)
				}
				fmt.Printf("Configuration %s is valid.\n", path)
				return nil
			},
		},
		{
			Name:  "config-generate",
			Usage: "Generate an example configuration file using the current flag values",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Value:   defaultExampleConfigPath,
					Usage:   "File path to write the generated configuration",
				},
				&cli.BoolFlag{
					Name:  "force",
					Usage: "Overwrite the output file if it already exists",
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				output := cmd.String("output")
				exampleConfig := flagParams.ConfigValues
				if err := validateConfigValues(exampleConfig); err != nil {
					return fmt.Errorf("generated configuration is invalid: %w", err)
				}
				if err := generateExampleConfigFile(output, exampleConfig, cmd.Bool("force")); err != nil {
					return err
				}
				fmt.Printf("Example configuration written to %s.\n", output)
				return nil
			},
		},
	}
	// Start service only for default action; version/help won't trigger this
	app.Action = func(ctx context.Context, cmd *cli.Command) error {
		if err := cliAction(); err != nil {
			return err
		}
		// Initialize service logger
		initializeLoggers(flagParams.ConfigValues)
		// Service starts!
		mapCTFService()
		return nil
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Printf("app.Run error: %s", err.Error())
		os.Exit(1)
	}
}
