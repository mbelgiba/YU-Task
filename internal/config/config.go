package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Address     string
	Environment string
	DBPath      string
	DevLogin    bool
}

func Load() (Config, error) {
	cfg := Config{
		Address:     value("YU_ADDR", ":8080"),
		Environment: value("YU_ENV", "development"),
		DBPath:      value("YU_DB_PATH", "./data/yu-tasks.db"),
	}
	devLogin, err := strconv.ParseBool(value("YU_DEV_LOGIN", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("YU_DEV_LOGIN must be true or false: %w", err)
	}
	cfg.DevLogin = devLogin
	if cfg.Environment == "production" && cfg.DevLogin {
		return Config{}, fmt.Errorf("YU_DEV_LOGIN is forbidden when YU_ENV=production")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
