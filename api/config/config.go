package config

import (
	"os"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server struct {
		Port        string
		ReadTimeout time.Duration
	}

	// Database configuration
	Database struct {
		ConnectionString string
	}

	// Environment
	Environment string

	// ChainID
	ChainID string

	// Storage
	Storage struct {
		Bucket string
	}
}

// New creates a new Config instance with values from environment variables
func New() *Config {
	config := &Config{}

	// Server configuration
	config.Server.Port = getEnv("PORT", "8080")
	config.Server.ReadTimeout = getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second)

	// Database configuration
	config.Database.ConnectionString = getEnv("DB_CONNECTION_STRING", "postgres://postgres:postgres@localhost:5432/core_indexer?sslmode=disable")

	// Environment
	config.Environment = getEnv("ENVIRONMENT", "local")

	// ChainID
	config.ChainID = getEnv("CHAIN_ID", "initiation-2")

	// GCS Bucket
	config.Storage.Bucket = config.ChainID + "-lcd-tx-responses"

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
