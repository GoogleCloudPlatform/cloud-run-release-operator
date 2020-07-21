package util

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKeyLogger struct{}

// The logger context key
var loggerKey contextKeyLogger

// ContextWithLogger returns a copy of the parent context that includes the
// logger.
func ContextWithLogger(ctx context.Context, logger *logrus.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext returns the logger from the context.
func LoggerFromContext(ctx context.Context) *logrus.Logger {
	logger := ctx.Value(loggerKey).(*logrus.Logger)
	return logger
}
