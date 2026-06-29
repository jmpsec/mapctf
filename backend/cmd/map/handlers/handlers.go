package handlers

import (
	"github.com/alexedwards/scs/v2"
	"github.com/jmpsec/mapctf/pkg/cache"
	"github.com/jmpsec/mapctf/pkg/challenges"
	"github.com/jmpsec/mapctf/pkg/chat"
	"github.com/jmpsec/mapctf/pkg/config"
	"github.com/jmpsec/mapctf/pkg/countries"
	"github.com/jmpsec/mapctf/pkg/i18n"
	"github.com/jmpsec/mapctf/pkg/logs"
	"github.com/jmpsec/mapctf/pkg/settings"
	"github.com/jmpsec/mapctf/pkg/teams"
	"github.com/jmpsec/mapctf/pkg/users"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"
)

const (
	// Default time format for loggers
	LoggerTimeFormat string = "2006-01-02T15:04:05.999Z07:00"
	// Default value for maximum size of log files in megabytes
	DefaultDebugMaxSize int = 25
	// Default value for maximum number of old log files to retain
	DefaultDebugMaxBackups int = 5
	// Default value for maximum number of days to retain old log files
	DefaultDebugMaxAge int = 10
	// Default value for compressing rotated log files
	DefaultDebugCompress bool = true
)

// HandlersMap to keep all handlers for the map service
type HandlersMap struct {
	ServiceName string
	DB          *gorm.DB
	RedisCache  *cache.RedisManager
	feeds       *respCache
	Teams       *teams.TeamManager
	Users       *users.UserManager
	Challenges  *challenges.ChallengeManager
	Settings    *settings.SettingsManager
	Countries   *countries.CountriesManager
	Logs        *logs.LogManager
	Chat        *chat.ChatManager
	Config      config.MapCTFConfiguration
	Sessions    *scs.SessionManager
	I18N        *i18n.Catalog
	DebugHTTP   *zerolog.Logger
}

// LumberjackConfig to keep configuration for rotating logs
type LumberjackConfig struct {
	// Maximum size in megabytes of the log file before it gets rotated
	MaxSize int
	// Maximum number of old log files to retain
	MaxBackups int
	// Maximum number of days to retain old log files based on the timestamp encoded in their filename
	MaxAge int
	// If the rotated log files should be compressed using gzip
	Compress bool
}

type HandlersOption func(*HandlersMap)

func WithServiceName(name string) HandlersOption {
	return func(h *HandlersMap) {
		h.ServiceName = name
	}
}

func WithDB(db *gorm.DB) HandlersOption {
	return func(h *HandlersMap) {
		h.DB = db
	}
}

func WithRedisCache(redis *cache.RedisManager) HandlersOption {
	return func(h *HandlersMap) {
		h.RedisCache = redis
	}
}

func WithConfig(cfg config.MapCTFConfiguration) HandlersOption {
	return func(h *HandlersMap) {
		h.Config = cfg
	}
}

func WithTeams(teams *teams.TeamManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Teams = teams
	}
}

func WithUsers(users *users.UserManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Users = users
	}
}

func WithChallenges(challenges *challenges.ChallengeManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Challenges = challenges
	}
}

func WithSettings(settings *settings.SettingsManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Settings = settings
	}
}

func WithLogs(logs *logs.LogManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Logs = logs
	}
}

func WithChat(chat *chat.ChatManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Chat = chat
	}
}

func WithCountries(countries *countries.CountriesManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Countries = countries
	}
}

func WithI18N(catalog *i18n.Catalog) HandlersOption {
	return func(h *HandlersMap) {
		h.I18N = catalog
	}
}

// createDebugHTTP to initialize the debug HTTP logger
func createDebugHTTP(filename string) (*zerolog.Logger, error) {
	zerolog.TimeFieldFormat = LoggerTimeFormat
	cfg := LumberjackConfig{
		MaxSize:    DefaultDebugMaxSize,
		MaxBackups: DefaultDebugMaxBackups,
		MaxAge:     DefaultDebugMaxAge,
		Compress:   DefaultDebugCompress,
	}
	z := zerolog.New(&lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})
	logger := z.With().Caller().Timestamp().Logger()
	return &logger, nil
}

func WithSessions(sessions *scs.SessionManager) HandlersOption {
	return func(h *HandlersMap) {
		h.Sessions = sessions
	}
}

func WithDebugHTTP(cfg *config.ConfigurationDebugHTTP) HandlersOption {
	return func(h *HandlersMap) {
		h.DebugHTTP = nil
		if cfg.Enabled {
			logger, err := createDebugHTTP(cfg.File)
			if err == nil {
				h.DebugHTTP = logger
			}
		}
	}
}

// CreateHandlersMap to create a new HandlersMap instance
func CreateHandlersMap(opts ...HandlersOption) *HandlersMap {
	h := &HandlersMap{}
	for _, opt := range opts {
		opt(h)
	}
	return h
}
