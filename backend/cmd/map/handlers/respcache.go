package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	// feedCacheL1TTL bounds how long an in-process copy of a feed response is
	// reused, coalescing a burst of simultaneous polls (feeds poll every 15s).
	feedCacheL1TTL = 4 * time.Second
	// feedCacheL2TTL is the Redis TTL for a feed response.
	feedCacheL2TTL     = 4 * time.Second
	feedCacheKeyPrefix = "mapctf:feed:"
)

// cacheResult is the unit shared by singleflight: an already-marshaled JSON body
// and the HTTP status to write with it. Errors are returned as non-200 results so
// they are never cached, but are still delivered to every concurrent waiter.
type cacheResult struct {
	status int
	body   []byte
}

type respCache struct {
	rdb *redis.Client
	l1  sync.Map // key -> *respCacheEntry
	sf  singleflight.Group
}

type respCacheEntry struct {
	body      []byte
	fetchedAt time.Time
}

func (e *respCacheEntry) fresh() bool { return time.Since(e.fetchedAt) < feedCacheL1TTL }

// feedCache lazily initializes a response cache backed by the configured Redis
// client. It returns nil when Redis is unavailable and no cache has been
// injected, in which case feed handlers build and respond without caching.
func (h *HandlersMap) feedCache() *respCache {
	if h.feeds != nil {
		return h.feeds
	}
	if h.RedisCache == nil || h.RedisCache.Client == nil {
		return nil
	}
	h.feeds = &respCache{rdb: h.RedisCache.Client}
	return h.feeds
}

func (c *respCache) l1Get(key string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	v, ok := c.l1.Load(key)
	if !ok {
		return nil, false
	}
	e := v.(*respCacheEntry)
	if !e.fresh() {
		return nil, false
	}
	return e.body, true
}

func (c *respCache) l1Put(key string, body []byte) {
	if c == nil {
		return
	}
	c.l1.Store(key, &respCacheEntry{body: body, fetchedAt: time.Now()})
}

func feedKey(name, uuid string) string { return feedCacheKeyPrefix + name + ":" + uuid }

// invalidateFeed drops the cached response for a feed. Call it from mutation
// paths (score, capture, chat post, admin CRUD) for snappier updates; the short
// TTL bounds staleness even without it.
func (h *HandlersMap) invalidateFeed(name, uuid string) {
	c := h.feedCache()
	if c == nil {
		return
	}
	key := feedKey(name, uuid)
	c.l1.Delete(key)
	if c.rdb != nil {
		c.rdb.Del(context.Background(), key)
	}
}

// serveCachedJSON serves a JSON response with cache-aside. On a cache hit the
// build function is not invoked, so the (expensive) DB queries inside it are
// skipped. build must return the HTTP status and the already-marshaled JSON
// body; only status 200 responses are cached. Concurrent misses for the same key
// are coalesced via singleflight and each waiter writes the shared result to its
// own ResponseWriter.
func (h *HandlersMap) serveCachedJSON(w http.ResponseWriter, key string, build func() (int, []byte)) {
	c := h.feedCache()
	if c == nil {
		status, body := build()
		writeStatusJSON(w, status, body)
		return
	}
	if body, ok := c.l1Get(key); ok {
		writeStatusJSON(w, http.StatusOK, body)
		return
	}
	v, _, _ := c.sf.Do(key, func() (any, error) {
		if body, ok := c.l1Get(key); ok {
			return cacheResult{http.StatusOK, body}, nil
		}
		if c.rdb != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			raw, rerr := c.rdb.Get(ctx, key).Bytes()
			cancel()
			if rerr == nil {
				c.l1Put(key, raw)
				return cacheResult{http.StatusOK, raw}, nil
			}
		}
		status, body := build()
		if status == http.StatusOK && c.rdb != nil {
			c.rdb.Set(context.Background(), key, body, feedCacheL2TTL)
		}
		if status == http.StatusOK {
			c.l1Put(key, body)
		}
		return cacheResult{status, body}, nil
	})
	cr := v.(cacheResult)
	writeStatusJSON(w, cr.status, cr.body)
}

func writeStatusJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set(ContentType, JSONApplicationUTF8)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// marshalJSON is a thin helper for feed build closures.
func marshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"error":"internal error"}`)
	}
	return b
}
