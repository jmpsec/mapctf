package settings

import (
	"database/sql"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) (*gorm.DB, *sql.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	return db, sqlDB
}

func newTestManager(t *testing.T) (*SettingsManager, *sql.DB) {
	t.Helper()

	db, sqlDB := newTestDB(t)
	m, err := CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	return m, sqlDB
}

func TestCreateSettingsManager(t *testing.T) {
	t.Run("nil db returns error", func(t *testing.T) {
		m, err := CreateSettingsManager(nil, "test-service")
		require.Nil(t, m)
		require.Error(t, err)
		require.Contains(t, err.Error(), "database connection cannot be nil")
	})

	t.Run("successfully migrates", func(t *testing.T) {
		db, sqlDB := newTestDB(t)
		defer func() { _ = sqlDB.Close() }()

		m, err := CreateSettingsManager(db, "test-service")
		require.NoError(t, err)
		require.NotNil(t, m)
		require.NotNil(t, m.DB)
		require.Equal(t, "test-service", m.Service)
	})

	t.Run("automigrate failure returns error", func(t *testing.T) {
		db, sqlDB := newTestDB(t)
		require.NoError(t, sqlDB.Close())

		m, err := CreateSettingsManager(db, "test-service")
		require.Nil(t, m)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to AutoMigrate table")
	})
}

func TestCreateExistsGetExistsGet(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	s := PlatformSetting{
		Name:        LoginEnabled,
		ValueType:   TypeBool,
		ValueBool:   true,
		Description: "toggle login",
		UUID:        "tenant-a",
	}
	require.NoError(t, m.Create(s))
	require.True(t, m.Exists(LoginEnabled, "tenant-a"))
	require.False(t, m.Exists(LoginEnabled, "tenant-b"))

	got, err := m.Get(LoginEnabled, "tenant-a")
	require.NoError(t, err)
	require.Equal(t, LoginEnabled, got.Name)
	require.Equal(t, "tenant-a", got.UUID)

	exists, setting := m.ExistsGet(LoginEnabled, "tenant-a")
	require.True(t, exists)
	require.Equal(t, got.ID, setting.ID)

	exists, setting = m.ExistsGet(LoginEnabled, "missing")
	require.False(t, exists)
	require.Equal(t, PlatformSetting{}, setting)
}

func TestGetMissingReturnsError(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	_, err := m.Get("does_not_exist", "tenant-a")
	require.Error(t, err)
}

func TestGetAllByUUID(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	require.NoError(t, m.Create(PlatformSetting{
		Name:      "tenant_a_only",
		ValueType: TypeString,
		UUID:      "tenant-a",
	}))
	require.NoError(t, m.Create(PlatformSetting{
		Name:      "tenant_b_only",
		ValueType: TypeString,
		UUID:      "tenant-b",
	}))

	allA, err := m.GetAll("tenant-a")
	require.NoError(t, err)
	require.Len(t, allA, 1)
	require.Equal(t, "tenant_a_only", allA[0].Name)
	require.Equal(t, "tenant-a", allA[0].UUID)
}

func TestInitializationCreatesDefaultsAndIsIdempotent(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	require.NoError(t, m.Initialization("tenant-a"))

	for name := range BooleanSettings {
		require.True(t, m.Exists(name, "tenant-a"), "missing boolean setting %s", name)
	}
	for name := range StringSettings {
		require.True(t, m.Exists(name, "tenant-a"), "missing string setting %s", name)
	}
	for name := range DateSettings {
		require.True(t, m.Exists(name, "tenant-a"), "missing date setting %s", name)
	}
	for name := range IntSettings {
		require.True(t, m.Exists(name, "tenant-a"), "missing int setting %s", name)
	}

	var firstCount int64
	require.NoError(t, m.DB.Model(&PlatformSetting{}).Where("uuid = ?", "tenant-a").Count(&firstCount).Error)
	require.NoError(t, m.Initialization("tenant-a"))
	var secondCount int64
	require.NoError(t, m.DB.Model(&PlatformSetting{}).Where("uuid = ?", "tenant-a").Count(&secondCount).Error)
	require.Equal(t, firstCount, secondCount)
}

func TestLogEvent(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	require.NoError(t, m.LogEvent(42, EventCreate, "alice", "tenant-a"))

	var logs []SettingLog
	require.NoError(t, m.DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, uint(42), logs[0].SettingID)
	require.Equal(t, EventCreate, logs[0].Event)
	require.Equal(t, "alice", logs[0].ChangedBy)
	require.Equal(t, "tenant-a", logs[0].UUID)
}

func TestNew(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	now := time.Now().UTC().Truncate(time.Second)
	tests := []struct {
		name      string
		valueType string
		value     any
		assertFn  func(*testing.T, PlatformSetting)
	}{
		{
			name:      "string",
			valueType: TypeString,
			value:     "hello",
			assertFn: func(t *testing.T, s PlatformSetting) {
				require.Equal(t, "hello", s.ValueString)
			},
		},
		{
			name:      "int",
			valueType: TypeInt,
			value:     7,
			assertFn: func(t *testing.T, s PlatformSetting) {
				require.Equal(t, 7, s.ValueInt)
			},
		},
		{
			name:      "bool",
			valueType: TypeBool,
			value:     true,
			assertFn: func(t *testing.T, s PlatformSetting) {
				require.True(t, s.ValueBool)
			},
		},
		{
			name:      "float",
			valueType: TypeFloat,
			value:     3.14,
			assertFn: func(t *testing.T, s PlatformSetting) {
				require.Equal(t, 3.14, s.ValueFloat)
			},
		},
		{
			name:      "date",
			valueType: TypeDate,
			value:     now,
			assertFn: func(t *testing.T, s PlatformSetting) {
				require.Equal(t, now, s.ValueDate)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := "setting_" + tc.name
			setting, err := m.New(name, tc.valueType, "desc", "tenant-a", tc.value)
			require.NoError(t, err)
			require.Equal(t, name, setting.Name)
			require.Equal(t, tc.valueType, setting.ValueType)
			require.Equal(t, "desc", setting.Description)
			require.Equal(t, "tenant-a", setting.UUID)
			tc.assertFn(t, setting)
		})
	}

	t.Run("invalid type returns error", func(t *testing.T) {
		_, err := m.New("bad", "unknown", "desc", "tenant-a", "x")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid value type")
	})

	t.Run("existing setting returns already exists error", func(t *testing.T) {
		existing := PlatformSetting{
			Name:      "dup",
			ValueType: TypeString,
			UUID:      "tenant-a",
		}
		require.NoError(t, m.Create(existing))

		_, err := m.New("dup", TypeString, "desc", "tenant-a", "x")
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})
}

func TestSaveCreatesAuditLog(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	s := PlatformSetting{
		Name:        CustomOrg,
		ValueType:   TypeString,
		ValueString: "Old",
		Description: "org",
		UUID:        "tenant-a",
	}
	require.NoError(t, m.Create(s))

	loaded, err := m.Get(CustomOrg, "tenant-a")
	require.NoError(t, err)

	loaded.ValueString = "New"
	require.NoError(t, m.Save(loaded, "bob"))

	var logs []SettingLog
	require.NoError(t, m.DB.Where("setting_id = ? AND event = ?", loaded.ID, EventUpdate).Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, "bob", logs[0].ChangedBy)
	require.Equal(t, "tenant-a", logs[0].UUID)
}

func TestChangeErrors(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	t.Run("missing setting returns get error", func(t *testing.T) {
		err := m.Change("missing", TypeString, "tenant-a", "x", "alice")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get setting")
	})

	t.Run("update succeeds", func(t *testing.T) {
		s := PlatformSetting{
			Name:        "some_setting",
			ValueType:   TypeString,
			ValueString: "before",
			UUID:        "tenant-a",
		}
		require.NoError(t, m.Create(s))

		err := m.Change("some_setting", TypeString, "tenant-a", "after", "alice")
		require.NoError(t, err)

		updated, getErr := m.Get("some_setting", "tenant-a")
		require.NoError(t, getErr)
		require.Equal(t, "after", updated.ValueString)
	})

	t.Run("invalid value type returns resolve column error", func(t *testing.T) {
		s := PlatformSetting{
			Name:        "bad_type_setting",
			ValueType:   TypeString,
			ValueString: "before",
			UUID:        "tenant-a",
		}
		require.NoError(t, m.Create(s))

		err := m.Change("bad_type_setting", "unknown-type", "tenant-a", "after", "alice")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to resolve setting value column")
	})
}

func TestChangeSuccessAndLogFailure(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	s := PlatformSetting{
		Name:        "with_value_column",
		ValueType:   TypeString,
		ValueString: "before",
		UUID:        "tenant-a",
	}
	require.NoError(t, m.Create(s))

	t.Run("successfully changes and logs", func(t *testing.T) {
		err := m.Change("with_value_column", TypeString, "tenant-a", "after", "alice")
		require.NoError(t, err)

		var logs []SettingLog
		require.NoError(t, m.DB.Where("event = ? AND changed_by = ?", EventUpdate, "alice").Find(&logs).Error)
		require.Len(t, logs, 1)
	})

	t.Run("logevent failure path", func(t *testing.T) {
		require.NoError(t, m.DB.Exec("DROP TABLE setting_logs").Error)

		err := m.Change("with_value_column", TypeString, "tenant-a", "after2", "alice")
		require.Error(t, err)
		require.Contains(t, err.Error(), "logEvent PlatformSetting")
	})
}

func TestCreateAndLogEventFailWhenDBClosed(t *testing.T) {
	m, sqlDB := newTestManager(t)
	require.NoError(t, sqlDB.Close())

	err := m.Create(PlatformSetting{Name: "x", UUID: "tenant-a"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "create PlatformSetting")

	err = m.LogEvent(1, EventUpdate, "alice", "tenant-a")
	require.Error(t, err)
	require.Contains(t, err.Error(), "create SettingLog")
}

func TestSaveFailsWhenDBClosed(t *testing.T) {
	m, sqlDB := newTestManager(t)

	s := PlatformSetting{
		Name:        "will_fail",
		ValueType:   TypeString,
		ValueString: "x",
		UUID:        "tenant-a",
	}
	require.NoError(t, m.Create(s))

	loaded, err := m.Get("will_fail", "tenant-a")
	require.NoError(t, err)

	require.NoError(t, sqlDB.Close())
	err = m.Save(loaded, "alice")
	require.Error(t, err)
	require.Contains(t, err.Error(), "save PlatformSetting")
}

func TestTypedGettersAndSetters(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	require.NoError(t, m.SetLoginEnabled(true, "alice", "tenant-a"))
	loginEnabled, err := m.GetLoginEnabled("tenant-a")
	require.NoError(t, err)
	require.True(t, loginEnabled)

	require.NoError(t, m.SetLoginStrongPasswords(true, "alice", "tenant-a"))
	loginStrongPasswords, err := m.GetLoginStrongPasswords("tenant-a")
	require.NoError(t, err)
	require.True(t, loginStrongPasswords)

	require.NoError(t, m.SetRegistrationEnabled(true, "alice", "tenant-a"))
	registrationEnabled, err := m.GetRegistrationEnabled("tenant-a")
	require.NoError(t, err)
	require.True(t, registrationEnabled)

	require.NoError(t, m.SetRegistrationNames(true, "alice", "tenant-a"))
	registrationNames, err := m.GetRegistrationNames("tenant-a")
	require.NoError(t, err)
	require.True(t, registrationNames)

	require.NoError(t, m.SetRegistrationEmails(true, "alice", "tenant-a"))
	registrationEmails, err := m.GetRegistrationEmails("tenant-a")
	require.NoError(t, err)
	require.True(t, registrationEmails)

	require.NoError(t, m.SetRegistrationType(1, "alice", "tenant-a"))
	registrationType, err := m.GetRegistrationType("tenant-a")
	require.NoError(t, err)
	require.Equal(t, 1, registrationType)

	require.NoError(t, m.SetRegistrationToken("invite-123", "alice", "tenant-a"))
	registrationToken, err := m.GetRegistrationToken("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "invite-123", registrationToken)

	require.NoError(t, m.SetScoringEnabled(true, "alice", "tenant-a"))
	scoringEnabled, err := m.GetScoringEnabled("tenant-a")
	require.NoError(t, err)
	require.True(t, scoringEnabled)

	require.NoError(t, m.SetScoringHints(true, "alice", "tenant-a"))
	scoringHints, err := m.GetScoringHints("tenant-a")
	require.NoError(t, err)
	require.True(t, scoringHints)

	require.NoError(t, m.SetScoringHelp(true, "alice", "tenant-a"))
	scoringHelp, err := m.GetScoringHelp("tenant-a")
	require.NoError(t, err)
	require.True(t, scoringHelp)

	require.NoError(t, m.SetGamePaused(true, "alice", "tenant-a"))
	gamePaused, err := m.GetGamePaused("tenant-a")
	require.NoError(t, err)
	require.True(t, gamePaused)

	require.NoError(t, m.SetGameStarted(true, "alice", "tenant-a"))
	gameStarted, err := m.GetGameStarted("tenant-a")
	require.NoError(t, err)
	require.True(t, gameStarted)

	require.NoError(t, m.SetGameStartTime(start, "alice", "tenant-a"))
	gameStartTime, err := m.GetGameStartTime("tenant-a")
	require.NoError(t, err)
	require.Equal(t, start, gameStartTime)

	require.NoError(t, m.SetGameEndTime(end, "alice", "tenant-a"))
	gameEndTime, err := m.GetGameEndTime("tenant-a")
	require.NoError(t, err)
	require.Equal(t, end, gameEndTime)

	require.NoError(t, m.SetCustomOrg("Acme", "alice", "tenant-a"))
	customOrg, err := m.GetCustomOrg("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "Acme", customOrg)

	require.NoError(t, m.SetCustomLogo("logo-michigan", "alice", "tenant-a"))
	customLogo, err := m.GetCustomLogo("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "logo-michigan", customLogo)

	require.NoError(t, m.SetLanguage("es", "alice", "tenant-a"))
	language, err := m.GetLanguage("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "es", language)

	require.NoError(t, m.SetLeaderboardLimit(25, "alice", "tenant-a"))
	leaderboardLimit, err := m.GetLeaderboardLimit("tenant-a")
	require.NoError(t, err)
	require.Equal(t, 25, leaderboardLimit)

	require.NoError(t, m.SetGameboardShowTeamMembers(true, "alice", "tenant-a"))
	gameboardShowTeamMembers, err := m.GetGameboardShowTeamMembers("tenant-a")
	require.NoError(t, err)
	require.True(t, gameboardShowTeamMembers)

	require.NoError(t, m.SetGameboardChatMaxLen(500, "alice", "tenant-a"))
	gameboardChatMaxLen, err := m.GetGameboardChatMaxLen("tenant-a")
	require.NoError(t, err)
	require.Equal(t, 500, gameboardChatMaxLen)

	// update path
	require.NoError(t, m.SetLoginEnabled(false, "alice", "tenant-a"))
	loginEnabled, err = m.GetLoginEnabled("tenant-a")
	require.NoError(t, err)
	require.False(t, loginEnabled)
}

func TestAllDeclaredSettingsHaveTypedAccessors(t *testing.T) {
	managerType := reflect.TypeOf(&SettingsManager{})

	var settingNames []string
	for name := range BooleanSettings {
		settingNames = append(settingNames, name)
	}
	for name := range StringSettings {
		settingNames = append(settingNames, name)
	}
	for name := range DateSettings {
		settingNames = append(settingNames, name)
	}
	for name := range IntSettings {
		settingNames = append(settingNames, name)
	}
	sort.Strings(settingNames)

	for _, settingName := range settingNames {
		methodSuffix := settingNameToMethodSuffix(settingName)

		setterName := "Set" + methodSuffix
		_, hasSetter := managerType.MethodByName(setterName)
		require.Truef(t, hasSetter, "missing setter %s for setting %s", setterName, settingName)

		getterName := "Get" + methodSuffix
		_, hasGetter := managerType.MethodByName(getterName)
		require.Truef(t, hasGetter, "missing getter %s for setting %s", getterName, settingName)
	}
}

func settingNameToMethodSuffix(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func TestTypedGettersTypeMismatch(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	require.NoError(t, m.Create(PlatformSetting{
		Name:        LoginEnabled,
		ValueType:   TypeString,
		ValueString: "not-bool",
		UUID:        "tenant-a",
	}))

	_, err := m.GetLoginEnabled("tenant-a")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected type")
}

func TestSaveFailsWhenLogInsertFails(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	s := PlatformSetting{
		Name:        "save_log_fail",
		ValueType:   TypeString,
		ValueString: "old",
		UUID:        "tenant-a",
	}
	require.NoError(t, m.Create(s))

	loaded, err := m.Get("save_log_fail", "tenant-a")
	require.NoError(t, err)
	loaded.ValueString = "new"

	require.NoError(t, m.DB.Exec("DROP TABLE setting_logs").Error)

	err = m.Save(loaded, "alice")
	require.Error(t, err)
	require.Contains(t, err.Error(), "logEvent PlatformSetting")
}

func TestSettingsCacheServesReadsAndInvalidatesOnWrite(t *testing.T) {
	// L1-only cache (no Redis) keeps the test network-free; the Redis L2 path
	// uses the same load/invalidate logic and is exercised in environments with
	// a reachable Redis.
	db, sqlDB := newTestDB(t)
	defer sqlDB.Close()
	m, err := CreateSettingsManager(db, "test-service")
	require.NoError(t, err)
	m.SetCache(nil)
	require.NoError(t, m.Initialization("tenant-a"))

	// Seed a value through the manager (write path invalidates the cache).
	require.NoError(t, m.SetLanguage("es", "alice", "tenant-a"))
	got, err := m.GetLanguage("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "es", got) // miss -> DB -> populate cache

	// Mutate the DB directly, bypassing the manager so no invalidation happens.
	require.NoError(t, db.Model(&PlatformSetting{}).
		Where("name = ? AND uuid = ?", Language, "tenant-a").
		Update("value_string", "fr").Error)

	// Cached read still returns the stale value, proving the cache served it.
	got, err = m.GetLanguage("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "es", got, "cached read should not reflect a bypassing DB write")

	// After invalidation the next read picks up the DB value.
	m.invalidate("tenant-a")
	got, err = m.GetLanguage("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "fr", got)

	// A manager write invalidates, so the new value is visible immediately.
	require.NoError(t, m.SetLanguage("de", "alice", "tenant-a"))
	got, err = m.GetLanguage("tenant-a")
	require.NoError(t, err)
	require.Equal(t, "de", got)
}
