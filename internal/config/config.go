package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address     string
	Environment string
	DBPath      string
	DevLogin    bool
}

func Load() (Config, error) {
	cfg := Config{
		Address:     value("YU_ADDR", "127.0.0.1:8080"),
		Environment: strings.ToLower(strings.TrimSpace(value("YU_ENV", "development"))),
		DBPath:      value("YU_DB_PATH", "./data/yu-tasks.db"),
	}
	if cfg.Environment != "development" && cfg.Environment != "production" {
		return Config{}, fmt.Errorf("YU_ENV must be development or production")
	}
	devLogin, err := strconv.ParseBool(value("YU_DEV_LOGIN", "false"))
	if err != nil {
		return Config{}, fmt.Errorf("YU_DEV_LOGIN must be true or false: %w", err)
	}
	cfg.DevLogin = devLogin
	if cfg.Environment == "production" && cfg.DevLogin {
		return Config{}, fmt.Errorf("YU_DEV_LOGIN is forbidden when YU_ENV=production")
	}
	if cfg.DevLogin {
		host, _, err := net.SplitHostPort(cfg.Address)
		ip := net.ParseIP(host)
		if err != nil || (host != "localhost" && (ip == nil || !ip.IsLoopback())) {
			return Config{}, fmt.Errorf("YU_DEV_LOGIN may only bind to localhost or a loopback IP")
		}
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
