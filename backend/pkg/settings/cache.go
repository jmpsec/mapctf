package settings

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	// settingsCacheL1TTL bounds how long an in-process copy is reused, coalescing
	// repeated reads within a single request burst (e.g. the gameboard handler
	// reads many settings at once).
	settingsCacheL1TTL = 2 * time.Second
	// settingsCacheL2TTL is the Redis TTL; a safety net since writes invalidate.
	settingsCacheL2TTL     = 60 * time.Second
	settingsCacheKeyPrefix = "mapctf:settings:"
)

// settingsCacheEntry holds both views of a UUID's settings: the ordered slice
// (for GetAll) and a name lookup map (for Get), built once when loaded.
type settingsCacheEntry struct {
	slice     []PlatformSetting
	byName    map[string]PlatformSetting
	fetchedAt time.Time
}

func newEntry(slice []PlatformSetting) *settingsCacheEntry {
	byName := make(map[string]PlatformSetting, len(slice))
	for _, s := range slice {
		byName[s.Name] = s
	}
	return &settingsCacheEntry{slice: slice, byName: byName, fetchedAt: time.Now()}
}

func (e *settingsCacheEntry) fresh() bool {
	return time.Since(e.fetchedAt) < settingsCacheL1TTL
}

// settingsCache is a per-UUID, two-layer (in-process L1 + Redis L2) cache-aside
// for platform settings. Settings are read on most requests but change only via
// admin writes, so reads hit the cache and writes invalidate it.
type settingsCache struct {
	rdb *redis.Client
	l1  sync.Map // uuid -> *settingsCacheEntry
	sf  singleflight.Group
}

func settingsCacheKey(uuid string) string { return settingsCacheKeyPrefix + uuid }

// SetCache enables Redis-backed caching for this manager. Passing nil disables
// it; reads then use only the in-process L1 over the DB.
func (m *SettingsManager) SetCache(rdb *redis.Client) {
	if rdb == nil {
		m.cache = &settingsCache{}
		return
	}
	m.cache = &settingsCache{rdb: rdb}
}

func (m *settingsCache) l1Get(uuid string) (*settingsCacheEntry, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m.l1.Load(uuid)
	if !ok {
		return nil, false
	}
	e := v.(*settingsCacheEntry)
	if !e.fresh() {
		return nil, false
	}
	return e, true
}

func (m *settingsCache) l1Put(uuid string, e *settingsCacheEntry) {
	if m == nil {
		return
	}
	m.l1.Store(uuid, e)
}

func (m *settingsCache) l1Delete(uuid string) {
	if m == nil {
		return
	}
	m.l1.Delete(uuid)
}

// loadCached returns the cached entry for a UUID, filling L1/Redis from the DB
// on miss. Cache errors never fail the read: any cache problem falls back to DB.
func (m *SettingsManager) loadCached(uuid string) (*settingsCacheEntry, error) {
	if e, ok := m.cacheL1Get(uuid); ok {
		return e, nil
	}
	if m.cache == nil {
		// Caching disabled: uncached DB read.
		return m.loadFromDB(uuid)
	}
	v, err, _ := m.cache.sf.Do(settingsCacheKey(uuid), func() (any, error) {
		if e, ok := m.cacheL1Get(uuid); ok {
			return e, nil
		}
		var slice []PlatformSetting
		// L2: Redis (best-effort; a miss or error falls through to the DB).
		if m.cache.rdb != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			raw, rerr := m.cache.rdb.Get(ctx, settingsCacheKey(uuid)).Bytes()
			cancel()
			if rerr == nil {
				if json.Unmarshal(raw, &slice) == nil {
					e := newEntry(slice)
					m.cacheL1Put(uuid, e)
					return e, nil
				}
			}
		}
		// DB load, then populate L2 (if enabled) and L1.
		var derr error
		slice, derr = m.getAllFromDB(uuid)
		if derr != nil {
			return nil, derr
		}
		if m.cache.rdb != nil {
			if data, merr := json.Marshal(slice); merr == nil {
				m.cache.rdb.Set(context.Background(), settingsCacheKey(uuid), data, settingsCacheL2TTL)
			}
		}
		e := newEntry(slice)
		m.cacheL1Put(uuid, e)
		return e, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*settingsCacheEntry), nil
}

func (m *SettingsManager) cacheL1Get(uuid string) (*settingsCacheEntry, bool) {
	if m.cache == nil {
		return nil, false
	}
	return m.cache.l1Get(uuid)
}

func (m *SettingsManager) cacheL1Put(uuid string, e *settingsCacheEntry) {
	if m.cache == nil {
		return
	}
	m.cache.l1Put(uuid, e)
}

// invalidate drops the cached settings for a UUID after a write.
func (m *SettingsManager) invalidate(uuid string) {
	if m.cache == nil {
		return
	}
	m.cache.l1Delete(uuid)
	if m.cache.rdb != nil {
		m.cache.rdb.Del(context.Background(), settingsCacheKey(uuid))
	}
}
