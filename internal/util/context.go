package util

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKey string

// Context keys.
const (
	contextKeyLogger = contextKey("logger")
)

// ContextWithLogger returns a copy of the parent context that includes the
// logger.
func ContextWithLogger(ctx context.Context, logger *logrus.Logger) context.Context {
	return context.WithValue(ctx, contextKeyLogger, logger)
}

// LoggerFromContext returns the logger from the context.
func LoggerFromContext(ctx context.Context) *logrus.Logger {
	logger := ctx.Value(contextKeyLogger).(*logrus.Logger)
	return logger
}
