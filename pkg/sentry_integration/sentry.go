package sentry_integration

import (
	"context"

	"github.com/getsentry/sentry-go"
)

func CaptureCurrentHubException(err error, level sentry.Level) {
	CaptureException(sentry.CurrentHub(), err, level)
}

func CaptureException(hub *sentry.Hub, err error, level sentry.Level) {
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		hub.CaptureException(err)
	})
}

func StartSentryTransaction(ctx context.Context, operation, description string) (*sentry.Span, context.Context) {
	transaction := sentry.StartTransaction(ctx, operation)
	transaction.Description = description
	return transaction, transaction.Context()
}

func StartSentrySpan(ctx context.Context, operation, description string) (*sentry.Span, context.Context) {
	span := sentry.StartSpan(ctx, operation)
	span.Description = description
	return span, span.Context()
}
