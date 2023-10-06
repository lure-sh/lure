package loggerctx

import (
	"context"

	"go.elara.ws/logger"
)

// loggerCtxKey is used as the context key for loggers
type loggerCtxKey struct{}

// With returns a copy of ctx containing log
func With(ctx context.Context, log logger.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, log)
}

// From attempts to get a logger from ctx. If ctx doesn't
// contain a logger, it returns a nop logger.
func From(ctx context.Context) logger.Logger {
	if val := ctx.Value(loggerCtxKey{}); val != nil {
		if log, ok := val.(logger.Logger); ok && log != nil {
			return log
		} else {
			return logger.NewNop()
		}
	} else {
		return logger.NewNop()
	}
}
