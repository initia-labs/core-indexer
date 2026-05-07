package logger

import (
	"context"
	"os"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Config holds the logger configuration
type Config struct {
	Component   string
	ChainID     string
	Environment string
	Level       zerolog.Level
}

// Init initializes the logger with the given configuration
func Init(cfg Config) *zerolog.Logger {
	// Set global log level
	zerolog.SetGlobalLevel(cfg.Level)

	// Use console writer for local environment
	var output zerolog.ConsoleWriter
	if cfg.Environment == "local" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
		}
		log = zerolog.New(output).With().Timestamp().Logger()
	} else {
		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	// Add context fields
	log = log.With().
		Str("component", cfg.Component).
		Str("chain_id", cfg.ChainID).
		Str("environment", cfg.Environment).
		Logger()

	// Set global context logger
	zerolog.Ctx(context.Background()).UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("component", cfg.Component).
			Str("chain_id", cfg.ChainID).
			Str("environment", cfg.Environment)
	})

	return &log
}

// Get returns the configured logger instance
func Get() *zerolog.Logger {
	return &log
}
