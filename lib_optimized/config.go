package lib

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Redis struct {
		Host        string `json:"host"`
		Port        int    `json:"port"`
		DB          int    `json:"db"`
		PoolSize    int    `json:"pool_size"`
		MaxIdle     int    `json:"max_idle"`
		MaxActive   int    `json:"max_active"`
		IdleTimeout int    `json:"idle_timeout"`
	} `json:"redis"`
}

func InitConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
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
