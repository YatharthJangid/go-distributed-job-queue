package lib

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfig_Valid(t *testing.T) {
	// Prepare temp JSON config file with Redis section
	cfgJSON := `{
		"redis": {
			"host": "localhost",
			"port": 6379,
			"db": 0,
			"pool_size": 10,
			"max_idle": 50,
			"max_active": 200,
			"idle_timeout": 30
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("unable to write temp config JSON: %v", err)
	}

	cfg, err := InitConfig(configPath)
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	if cfg.Redis.Host != "localhost" {
		t.Errorf("expected Redis.Host=localhost, got %s", cfg.Redis.Host)
	}
	if cfg.Redis.PoolSize != 10 {
		t.Errorf("expected Redis.PoolSize=10, got %d", cfg.Redis.PoolSize)
	}
	if cfg.Redis.MaxIdle != 50 || cfg.Redis.MaxActive != 200 {
		t.Errorf("unexpected Redis pool sizes MaxIdle=%d MaxActive=%d", cfg.Redis.MaxIdle, cfg.Redis.MaxActive)
	}
}

func TestInitConfig_MissingFile(t *testing.T) {
	cfg, err := InitConfig("non_existent.json")
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
	if cfg != nil {
		t.Errorf("expected empty config on error, got %#v", cfg)
	}
}

func TestInitConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")
	badJSON := `{ "redis": { "host": "localhost",, } }` // invalid JSON

	if err := os.WriteFile(configPath, []byte(badJSON), 0644); err != nil {
		t.Fatalf("unable to write invalid JSON: %v", err)
	}

	cfg, err := InitConfig(configPath)
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	if cfg != nil {
		t.Errorf("expected empty config on JSON error, got %#v", cfg)
	}
}

func TestInitConfig_MissingRedisSection(t *testing.T) {
	// JSON without "redis" key - should get defaults
	cfgJSON := `{}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "no_redis.json")

	if err := os.WriteFile(configPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := InitConfig(configPath)
	if err != nil {
		t.Fatalf("InitConfig failed on empty JSON: %v", err)
	}

	// All defaults should apply
	if cfg.Redis.PoolSize != 10 || cfg.Redis.MaxIdle != 50 {
		t.Errorf("Defaults not applied: %+v", cfg.Redis)
	}
}

func TestInitConfig_PartialDefaults(t *testing.T) {
	// JSON with some missing fields - test partial defaults
	cfgJSON := `{
		"redis": {
			"host": "localhost",
			"pool_size": 5
		}
	}`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.json")

	if err := os.WriteFile(configPath, []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := InitConfig(configPath)
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Verify override + defaults
	if cfg.Redis.PoolSize != 5 { // From JSON
		t.Errorf("PoolSize override failed: got %d", cfg.Redis.PoolSize)
	}
	if cfg.Redis.MaxIdle != 50 { // Default applied
		t.Errorf("MaxIdle default failed: got %d", cfg.Redis.MaxIdle)
	}
}
