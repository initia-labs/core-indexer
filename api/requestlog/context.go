package requestlog

import (
	"context"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type loggerKey struct{}

// WithCaller records caller-supplied monitoring metadata, not authenticated identity.
func WithCaller(ctx context.Context, indexerPath, scanPath string) context.Context {
	scanPath = strings.SplitN(scanPath, "?", 2)[0]
	caller := "direct"
	if scanPath != "" {
		caller = "scan-api"
	}
	logger := log.With().Str("caller", caller).Str("indexer_path", indexerPath)
	if scanPath != "" {
		logger = logger.Str("scan_api_path", scanPath)
	}
	result := logger.Logger()
	return context.WithValue(ctx, loggerKey{}, &result)
}

func FromContext(ctx context.Context) *zerolog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*zerolog.Logger); ok {
		return logger
	}
	return &log.Logger
}
