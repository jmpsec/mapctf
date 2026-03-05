package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/goredisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jmpsec/mapctf/cmd/map/handlers"
	"github.com/jmpsec/mapctf/pkg/backend"
	"github.com/jmpsec/mapctf/pkg/cache"
	"github.com/jmpsec/mapctf/pkg/challenges"
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
	serviceName = projectName + "-map"
	// Service description
	serviceDescription = "Map service for " + projectName
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
	loginPath string = "/login"
	// Default endpoint to handle Logout
	logoutPath string = "/logout"
	// Default endpoint to handle HTTP(500) errors
	errorPath string = "/error"
	// Default endpoint to handle Forbidden(403) errors
	forbiddenPath string = "/forbidden"
	// Default endpoint for favicon
	faviconPath string = "/favicon.ico"
	// Map gameboard path
	mapGameboardPath = "/gameboard"
	// Registration path
	registrationPath = "/registration"
	// Countdown path
	countdownPath = "/countdown"
	// Rules path
	rulesPath = "/rules"
	// Admin path
	adminPath = "/admin"
	// JSON data path
	jsonPath = "/json"
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
	flags = config.InitGameFlags(&flagParams)
	// Check if UUID was set, otherwise generate a random one for this instance
	if flagParams.ConfigValues.Map.UUID == "" {
		flagParams.ConfigValues.Map.UUID = config.GenUUID()
	}
}

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
	teamsMgr, err := teams.CreateTeams(db.Conn)
	if err != nil {
		log.Fatal().Msgf("Failed to initialize teams: %v", err)
	}
	// User Manager
	log.Info().Msg("Initialize users")
	usersMgr, err := users.CreateUserManager(db.Conn, &flagParams.ConfigValues.JWT)
	if err != nil {
		log.Fatal().Msgf("Failed to initialize users: %v", err)
	}
	// Challenge Manager
	log.Info().Msg("Initialize challenges")
	challengesMgr, err := challenges.CreateChallengeManager(db.Conn)
	if err != nil {
		log.Fatal().Msgf("Failed to initialize challenges: %v", err)
	}
	// Session manager
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.IdleTimeout = 30 * time.Minute
	sessionManager.Cookie.Name = "session_id"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Secure = true
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.Persist = true
	sessionManager.Store = goredisstore.New(redis.Client)
	// Handlers
	log.Info().Msg("Initializing handlers")
	handlersMap := handlers.CreateHandlersMap(
		handlers.WithServiceName(serviceName),
		handlers.WithDB(db.Conn),
		handlers.WithRedisCache(redis),
		handlers.WithConfig(flagParams.ConfigValues),
		handlers.WithTeams(teamsMgr),
		handlers.WithUsers(usersMgr),
		handlers.WithChallenges(challengesMgr),
		handlers.WithSessions(sessionManager),
		handlers.WithDebugHTTP(&flagParams.ConfigValues.DebugHTTP),
	)
	// Router
	log.Info().Msg("Initializing router")
	// Create chi router for map
	muxMap := chi.NewRouter()
	// Middleware
	muxMap.Use(middleware.RequestID)
	muxMap.Use(middleware.RealIP)
	muxMap.Use(middleware.Logger)
	muxMap.Use(middleware.Recoverer)
	muxMap.Use(middleware.Timeout(30 * time.Second))
	muxMap.Use(sessionManager.LoadAndSave)
	// Root
	muxMap.Get(rootPath, handlersMap.RootHandler)
	// Health
	muxMap.Get(healthPath, handlersMap.HealthHandler)
	// Error
	muxMap.Get(errorPath, handlersMap.ErrorHandler)
	// Forbidden
	muxMap.Get(forbiddenPath, handlersMap.ForbiddenHandler)
	// Favicon
	muxMap.Get(faviconPath, handlersMap.FaviconHandler)
	// Static files
	muxMap.Handle("/static/*", http.StripPrefix("/static/", ContentTypeByExtension(http.FileServer(http.Dir(flagParams.ConfigValues.Map.StaticDir)))))
	// HTTP map routes
	muxMap.Route("/{uuid}", func(r chi.Router) {
		// Public routes (no authentication required)
		r.Get(rootPath, handlersMap.IndexTemplateHandler)
		r.Get(errorPath, handlersMap.ErrorHandler)
		r.Get(loginPath, handlersMap.LoginHandler)
		r.Get(registrationPath, handlersMap.RegistrationTemplateHandler)
		r.Get(countdownPath, handlersMap.CountdownTemplateHandler)
		r.Get(rulesPath, handlersMap.RulesTemplateHandler)
		r.Post(loginPath, handlersMap.LoginPOSTHandler)
		r.Post(registrationPath, handlersMap.RegistrationPOSTHandler)
		r.Post(logoutPath, handlersMap.LogoutPOSTHandler)
		// Protected routes group (require authentication)
		r.Group(func(r chi.Router) {
			r.Use(handlersMap.RequireAuth)
			// Protected gameboard routes
			r.Get(mapGameboardPath, handlersMap.GameboardTemplateHandler)
			// Protected admin routes
			r.Route(adminPath, func(r chi.Router) {
				r.Use(handlersMap.RequireAdmin)
			})
		})
	})
	// Launch HTTP server for map
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
			Addr:    serviceListener,
			Handler: muxMap,
			// TODO: Make timeouts configurable
			ReadTimeout:       10 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			TLSConfig:         cfg,
			TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
		log.Info().Msgf("%s v%s - HTTPS listening %s/%s/", serviceName, buildVersion, serviceListener, flagParams.ConfigValues.Map.UUID)
		log.Info().Msgf("%s - commit=%s - build date=%s", serviceName, buildCommit, buildDate)
		if err := srv.ListenAndServeTLS(flagParams.ConfigValues.TLS.CertificateFile, flagParams.ConfigValues.TLS.KeyFile); err != nil {
			log.Fatal().Msgf("ListenAndServeTLS: %v", err)
		}
	} else {
		log.Info().Msgf("%s v%s - HTTP listening %s/%s/", serviceName, buildVersion, serviceListener, flagParams.ConfigValues.Map.UUID)
		log.Info().Msgf("%s - commit=%s - build date=%s", serviceName, buildCommit, buildDate)
		if err := http.ListenAndServe(serviceListener, muxMap); err != nil {
			log.Fatal().Msgf("ListenAndServe: %v", err)
		}
	}
}

// Middleware to set Content-Type header based on file extension for static files
func ContentTypeByExtension(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ext := filepath.Ext(r.URL.Path); ext != "" {
			if contentType := mime.TypeByExtension(ext); contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
		}
		next.ServeHTTP(w, r)
	})
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
		{
			Name:     "create-admin-user",
			Category: "configuration",
			Usage:    "Create or reset password for an admin user for the specified entity ID",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "username",
					Aliases: []string{"u"},
					Usage:   "Username for the admin user. If the user already exists, the password will be reset",
					Value:   "admin",
				},
				&cli.StringFlag{
					Name:    "password",
					Aliases: []string{"p"},
					Usage:   "Password for the admin user. If not provided, a random password will be generated",
					Value:   "",
				},
				&cli.StringFlag{
					Name:    "uuid",
					Aliases: []string{"u"},
					Usage:   "UUID for the admin user",
					Value:   users.NoUUID,
				},
			},
			Action: func(ctx context.Context, cmd *cli.Command) error {
				// Load configuration
				if err := cliAction(); err != nil {
					return err
				}

				// Initialize service logger
				initializeLoggers(flagParams.ConfigValues)
				// Initialize database connection
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
				// Initialize users manager
				usersMgr, err := users.CreateUserManager(db.Conn, &flagParams.ConfigValues.JWT)
				if err != nil {
					return fmt.Errorf("failed to initialize users manager: %w", err)
				}
				username := cmd.String("username")
				password := cmd.String("password")
				uuid := cmd.String("uuid")
				if uuid == "" {
					uuid = users.NoUUID
				}
				// Generate random password if not provided
				if password == "" {
					password = GenerateRandomPassword(12)
				}
				// Check if user exists
				if usersMgr.Exists(username, uuid) {
					// User exists, reset password
					log.Info().Msgf("User '%s' already exists for UUID %s, resetting password...", username, uuid)
					usersMgr.SetPassword(username, password, uuid)
					fmt.Printf("Password reset successfully for admin user '%s' (UUID %s).\n", username, uuid)
				} else {
					// User doesn't exist, create it
					log.Info().Msgf("Creating new admin user '%s' for UUID %s...", username, uuid)
					user, err := usersMgr.New(username, password, "", username, true, false, uuid, users.NoTeamID)
					if err != nil {
						return fmt.Errorf("failed to create admin user: %w", err)
					}
					// Save user to database
					if err := usersMgr.Create(user); err != nil {
						return fmt.Errorf("failed to save admin user: %w", err)
					}
					fmt.Printf("Admin user '%s' created successfully for UUID %s.\n", username, uuid)
				}
				// Print password
				fmt.Printf("\nPassword: %s\n", password)
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
