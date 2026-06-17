package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBDSN            string
	HTTPPort         string
	LogLevel         string
	TTLCleanInterval time.Duration
	RolloutBatchSize int
	ReportsDir       string
	RunMigrations    bool
}

func Load() (*Config, error) {
	cfg := &Config{
		DBDSN:      os.Getenv("DB_DSN"),
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		ReportsDir: getEnv("REPORTS_DIR", "./reports"),
	}

	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("config: DB_DSN is required")
	}

	interval, err := getEnvDuration("TTL_CLEAN_INTERVAL", time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.TTLCleanInterval = interval

	batch, err := getEnvInt("ROLLOUT_BATCH_SIZE", 1000)
	if err != nil {
		return nil, err
	}
	cfg.RolloutBatchSize = batch

	cfg.RunMigrations = getEnvBool("RUN_MIGRATIONS", true)

	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", key, err)
	}
	return n, nil
}

func getEnvDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a duration (e.g. 30s, 5m): %w", key, err)
	}
	return d, nil
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
