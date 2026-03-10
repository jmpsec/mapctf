package settings

import (
	"database/sql"
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
	m, err := CreateSettingsManager(db, "test-service", "tenant-a")
	require.NoError(t, err)
	return m, sqlDB
}

func TestCreateSettingsManager(t *testing.T) {
	t.Run("nil db returns error", func(t *testing.T) {
		m, err := CreateSettingsManager(nil, "test-service", "tenant-a")
		require.Nil(t, m)
		require.Error(t, err)
		require.Contains(t, err.Error(), "database connection cannot be nil")
	})

	t.Run("successfully migrates", func(t *testing.T) {
		db, sqlDB := newTestDB(t)
		defer func() { _ = sqlDB.Close() }()

		m, err := CreateSettingsManager(db, "test-service", "tenant-a")
		require.NoError(t, err)
		require.NotNil(t, m)
		require.NotNil(t, m.DB)
		require.Equal(t, "test-service", m.Service)
		require.Equal(t, "tenant-a", m.UUID)
	})

	t.Run("automigrate failure returns error", func(t *testing.T) {
		db, sqlDB := newTestDB(t)
		require.NoError(t, sqlDB.Close())

		m, err := CreateSettingsManager(db, "test-service", "tenant-a")
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

	t.Run("update failure returns error", func(t *testing.T) {
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
		require.Contains(t, err.Error(), "LogEvent PlatformSetting")
	})
}

func TestCreateAndLogEventFailWhenDBClosed(t *testing.T) {
	m, sqlDB := newTestManager(t)
	require.NoError(t, sqlDB.Close())

	err := m.Create(PlatformSetting{Name: "x", UUID: "tenant-a"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Create PlatformSetting")

	err = m.LogEvent(1, EventUpdate, "alice", "tenant-a")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Create SettingLog")
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
	require.Contains(t, err.Error(), "Save PlatformSetting")
}

func TestTypedGettersAndSetters(t *testing.T) {
	m, sqlDB := newTestManager(t)
	defer func() { _ = sqlDB.Close() }()

	start := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	require.NoError(t, m.SetLoginEnabled(true, "alice"))
	loginEnabled, err := m.GetLoginEnabled()
	require.NoError(t, err)
	require.True(t, loginEnabled)

	require.NoError(t, m.SetRegistrationEnabled(true, "alice"))
	registrationEnabled, err := m.GetRegistrationEnabled()
	require.NoError(t, err)
	require.True(t, registrationEnabled)

	require.NoError(t, m.SetScoringEnabled(true, "alice"))
	scoringEnabled, err := m.GetScoringEnabled()
	require.NoError(t, err)
	require.True(t, scoringEnabled)

	require.NoError(t, m.SetGamePaused(true, "alice"))
	gamePaused, err := m.GetGamePaused()
	require.NoError(t, err)
	require.True(t, gamePaused)

	require.NoError(t, m.SetGameStarted(true, "alice"))
	gameStarted, err := m.GetGameStarted()
	require.NoError(t, err)
	require.True(t, gameStarted)

	require.NoError(t, m.SetGameStartTime(start, "alice"))
	gameStartTime, err := m.GetGameStartTime()
	require.NoError(t, err)
	require.Equal(t, start, gameStartTime)

	require.NoError(t, m.SetGameEndTime(end, "alice"))
	gameEndTime, err := m.GetGameEndTime()
	require.NoError(t, err)
	require.Equal(t, end, gameEndTime)

	require.NoError(t, m.SetCustomOrg("Acme", "alice"))
	customOrg, err := m.GetCustomOrg()
	require.NoError(t, err)
	require.Equal(t, "Acme", customOrg)

	// update path
	require.NoError(t, m.SetLoginEnabled(false, "alice"))
	loginEnabled, err = m.GetLoginEnabled()
	require.NoError(t, err)
	require.False(t, loginEnabled)
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

	_, err := m.GetLoginEnabled()
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
	require.Contains(t, err.Error(), "LogEvent PlatformSetting")
}
