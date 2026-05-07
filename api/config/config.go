package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server struct {
		Port         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}

	// Database configuration
	Database struct {
		ConnectionString string
		MaxOpenConns     int
		MaxIdleConns     int
		ConnMaxLifetime  time.Duration
		ConnMaxIdleTime  time.Duration
	}

	// Repository configuration
	Repository struct {
		CountQueryTimeout time.Duration
	}

	// Environment
	Environment string

	// ChainID
	ChainID string

	// Storage
	Storage struct {
		Buckets              []string
		TxResponseCacheBytes int64
		TxMaxWorkers         int
	}

	// Observability
	Observability struct {
		RuntimeMetricsEnabled bool
	}
}

// New creates a new Config instance with values from environment variables
func New() *Config {
	config := &Config{}

	// Server configuration
	config.Server.Port = getEnv("PORT", "8080")
	config.Server.ReadTimeout = getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second)
	config.Server.WriteTimeout = getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second)
	config.Server.IdleTimeout = getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second)

	// Database configuration
	config.Database.ConnectionString = getEnv("DB_CONNECTION_STRING_RO", "postgres://postgres:postgres@localhost:5432/core_indexer?sslmode=disable")
	config.Database.MaxOpenConns = getIntEnv("DB_MAX_OPEN_CONNS", 10)
	config.Database.MaxIdleConns = getIntEnv("DB_MAX_IDLE_CONNS", 5)
	config.Database.ConnMaxLifetime = getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	config.Database.ConnMaxIdleTime = getDurationEnv("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)

	// Repository configuration
	config.Repository.CountQueryTimeout = getDurationEnv("REPOSITORY_COUNT_QUERY_TIMEOUT", 5*time.Second)

	// Environment
	config.Environment = getEnv("ENVIRONMENT", "local")

	// ChainID
	config.ChainID = getEnv("CHAIN_ID", "initiation-2")

	// GCS Buckets
	config.Storage.Buckets = strings.Split(getEnv("BUCKETS", ""), ",")
	config.Storage.TxResponseCacheBytes = getBytesEnv("TX_RESPONSE_CACHE_BYTES", 64*1024*1024)
	config.Storage.TxMaxWorkers = getIntEnv("TX_RESPONSE_MAX_WORKERS", 10)

	// Observability
	config.Observability.RuntimeMetricsEnabled = getBoolEnv("ENABLE_RUNTIME_METRICS", false)

	return config
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getDurationEnv gets a duration environment variable or returns a default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// getIntEnv gets an integer environment variable or returns a default value.
func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// getBoolEnv gets a boolean environment variable or returns a default value.
func getBoolEnv(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// getBytesEnv parses byte values with optional KiB/MiB/GiB or KB/MB/GB suffixes.
func getBytesEnv(key string, defaultValue int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	unit := int64(1)
	number := value
	lower := strings.ToLower(value)

	units := []struct {
		suffix     string
		multiplier int64
	}{
		{suffix: "gib", multiplier: 1024 * 1024 * 1024},
		{suffix: "gb", multiplier: 1000 * 1000 * 1000},
		{suffix: "mib", multiplier: 1024 * 1024},
		{suffix: "mb", multiplier: 1000 * 1000},
		{suffix: "kib", multiplier: 1024},
		{suffix: "kb", multiplier: 1000},
		{suffix: "b", multiplier: 1},
	}

	for _, candidate := range units {
		if strings.HasSuffix(lower, candidate.suffix) {
			unit = candidate.multiplier
			number = strings.TrimSpace(value[:len(value)-len(candidate.suffix)])
			break
		}
	}

	parsed, err := strconv.ParseInt(number, 10, 64)
	if err != nil || parsed < 0 {
		return defaultValue
	}
	return parsed * unit
}
