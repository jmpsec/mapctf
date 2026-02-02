package main

import (
	"context"
	"crypto/tls"
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
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/jmpsec/mapctf/pkg/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v3"
)

const (
	// Project name
	projectName = "mapCTF"
	// Service name
	serviceName = projectName + "-api"
	// Service description
	serviceDescription = "API service for " + projectName
	// Application description
	appDescription = serviceDescription + " a map-based CTF platform"
	// Default time format for loggers
	LoggerTimeFormat string = "2006-01-02T15:04:05.999Z07:00"
	// Default path used when generating example configs
	defaultExampleConfigPath = "config/mapctf.example.yaml"
	// Default path for service configuration file
	defaultConfigPath = "config/mapctf.yaml"
)

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
	// API users path
	apiUsersPath = "/users"
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

func configFileFromCommand(cmd *cli.Command) string {
	if file := cmd.String("file"); file != "" {
		return file
	}
	if flagParams.ConfigFile != "" {
		return flagParams.ConfigFile
	}
	return defaultConfigPath
}

// Initialization code
func init() {
	// Initialize CLI flags using the config package
	flags = config.InitMapFlags(&flagParams)
}

/*
POST   /api/auth/login          - User login
GET    /api/auth/logout         - User logout
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
	// Backend
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
	// Cache
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
	// Team Manager
	log.Info().Msg("Initialize teams")
	teamsMgr := teams.CreateTeams(db.Conn)
	// User Manager
	log.Info().Msg("Initialize users")
	usersMgr := users.CreateUserManager(db.Conn)
	// Handlers
	log.Info().Msg("Initializing handlers")
	handlersCTF := handlers.CreateHandlersAPI(
		handlers.WithDB(db.Conn),
		handlers.WithRedisCache(redis),
		handlers.WithConfig(flagParams.ConfigValues),
		handlers.WithTeams(teamsMgr),
		handlers.WithUsers(usersMgr), // User manager to be added
		handlers.WithDebugHTTP(&flagParams.ConfigValues.DebugHTTP),
	)
	// Router
	log.Info().Msg("Initializing router")
	// Create router for API
	muxAPI := http.NewServeMux()
	// Root
	muxAPI.HandleFunc("GET /", handlersCTF.RootHandler)
	// Testing
	muxAPI.HandleFunc("GET "+healthPath, handlersCTF.HealthHandler)
	// Error
	muxAPI.HandleFunc("GET "+errorPath, handlersCTF.ErrorHandler)
	// Forbidden
	muxAPI.HandleFunc("GET "+forbiddenPath, handlersCTF.ForbiddenHandler)
	// Check status
	muxAPI.HandleFunc("GET "+_apiPath(checksNoAuthPath), handlersCTF.CheckHandlerNoAuth)
	// Login
	muxAPI.HandleFunc("POST "+_apiPath(loginPath), handlersCTF.LoginHandler)
	// Logout
	muxAPI.HandleFunc("GET "+_apiPath(logoutPath), handlersCTF.LogoutHandler)
	// Launch HTTP server for api
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
	if err := config.ValidateConfigValues(flagParams.ConfigValues); err != nil {
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
	// Customize command help template to hide global options
	cli.CommandHelpTemplate = CommandHelpTemplateString

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
			Name:     "config-validate",
			Category: "configuration",
			Usage:    "Validate a MapCTF configuration file",
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
				if err := config.ValidateConfigValues(cfg); err != nil {
					return fmt.Errorf("configuration %s is invalid: %w", path, err)
				}
				fmt.Printf("Configuration %s is valid.\n", path)
				return nil
			},
		},
		{
			Name:     "config-generate",
			Category: "configuration",
			Usage:    "Generate an example configuration file using the current flag values",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Value:   defaultExampleConfigPath,
					Usage:   "File path to write the generated configuration",
				},
				&cli.BoolFlag{
					Name:    "force",
					Aliases: []string{"f"},
					Usage:   "Overwrite the output file if it already exists",
					Value:   false,
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				output := cmd.String("output")
				exampleConfig := flagParams.ConfigValues
				if err := config.ValidateConfigValues(exampleConfig); err != nil {
					return fmt.Errorf("generated configuration is invalid: %w", err)
				}
				if err := config.GenConfigFile(output, exampleConfig, cmd.Bool("force")); err != nil {
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
