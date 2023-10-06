package loggerctx

import (
	"context"

	"go.elara.ws/logger"
)

type loggerCtxKey struct{}

func With(ctx context.Context, log logger.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, log)
}

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
