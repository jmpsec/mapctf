package cache

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/jmpsec/mapctf/pkg/config"
)

func setupTestRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	t.Cleanup(func() {
		mr.Close()
	})
	return mr
}

func TestConstants(t *testing.T) {
	if RedisKey != "redis" {
		t.Errorf("Expected redis, got %q", RedisKey)
	}
}

func TestCreateRedisManager(t *testing.T) {
	mr := setupTestRedis(t)

	cfg := config.ConfigurationRedis{
		Host:     mr.Host(),
		Port:     mr.Port(),
		Password: "",
		DB:       0,
	}

	manager, err := CreateRedisManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create Redis manager: %v", err)
	}

	if manager == nil || manager.Client == nil || manager.Config == nil {
		t.Fatal("Expected fully initialized RedisManager")
	}
}

func TestCreateRedisManagerWithInvalidConfig(t *testing.T) {
	cfg := config.ConfigurationRedis{
		Host: "invalid-host.example.com",
		Port: "6379",
	}

	_, err := CreateRedisManager(cfg)
	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

func TestGetRedisWithHostPort(t *testing.T) {
	mr := setupTestRedis(t)

	cfg := config.ConfigurationRedis{
		Host: mr.Host(),
		Port: mr.Port(),
	}

	manager := &RedisManager{Config: &cfg}
	client := manager.GetRedis()

	if client == nil {
		t.Fatal("Expected non-nil Redis client")
	}

	if err := client.Ping(client.Context()).Err(); err != nil {
		t.Errorf("Failed to ping Redis: %v", err)
	}
}

func TestGetRedisWithConnectionString(t *testing.T) {
	mr := setupTestRedis(t)

	cfg := config.ConfigurationRedis{
		ConnectionString: "redis://" + mr.Addr(),
	}

	manager := &RedisManager{Config: &cfg}
	client := manager.GetRedis()

	if client == nil || client.Ping(client.Context()).Err() != nil {
		t.Fatal("Expected valid Redis client with connection string")
	}
}

func TestCheckSuccess(t *testing.T) {
	mr := setupTestRedis(t)

	cfg := config.ConfigurationRedis{
		Host: mr.Host(),
		Port: mr.Port(),
	}

	manager, _ := CreateRedisManager(cfg)

	if err := manager.Check(); err != nil {
		t.Errorf("Expected successful check: %v", err)
	}
}

func TestCheckFailure(t *testing.T) {
	mr := setupTestRedis(t)

	cfg := config.ConfigurationRedis{
		Host: mr.Host(),
		Port: mr.Port(),
	}

	manager, _ := CreateRedisManager(cfg)
	mr.Close()

	if err := manager.Check(); err == nil {
		t.Error("Expected error when checking closed connection")
	}
}

func TestRedisOperations(t *testing.T) {
	mr := setupTestRedis(t)

	manager, _ := CreateRedisManager(config.ConfigurationRedis{
		Host: mr.Host(),
		Port: mr.Port(),
	})

	ctx := manager.Client.Context()

	if err := manager.Client.Set(ctx, "key", "value", 0).Err(); err != nil {
		t.Errorf("Failed to set: %v", err)
	}

	val, err := manager.Client.Get(ctx, "key").Result()
	if err != nil || val != "value" {
		t.Errorf("Failed to get: %v", err)
	}
}
