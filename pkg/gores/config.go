package gores

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	Redis struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		URL         string `json:"url"`
		DB          int    `json:"db"`
		PoolSize    int    `json:"pool_size"`
		MaxIdle     int    `json:"max_idle"`
		MaxActive   int    `json:"max_active"`
		IdleTimeout int    `json:"idle_timeout"`
	} `json:"redis"`
}

func InitConfig(path string) (*Config, error) {
	var cfg Config

	// Try reading from file first
	data, err := ioutil.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	} else if os.Getenv("REDIS_URL") == "" {
		// Only fail if file is missing AND there is no REDIS_URL
		return nil, err
	}

	// Override with environment variable if present
	if url := os.Getenv("REDIS_URL"); url != "" {
		cfg.Redis.URL = url
	}

	if cfg.Redis.PoolSize == 0 {
		cfg.Redis.PoolSize = 10
	}
	if cfg.Redis.MaxIdle == 0 {
		cfg.Redis.MaxIdle = 50
	}
	if cfg.Redis.MaxActive == 0 {
		cfg.Redis.MaxActive = 200
	}
	return &cfg, nil
}
