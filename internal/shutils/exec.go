package shutils

import (
	"context"
	"time"

	"mvdan.cc/sh/v3/interp"
)

type ExecFuncs map[string]func(interp.HandlerContext, []string) uint8

func (ef ExecFuncs) ExecHandler(ctx context.Context, args []string) error {
	name := args[0]

	if fn, ok := ef[name]; ok {
		hctx := interp.HandlerCtx(ctx)
		ec := fn(hctx, args)
		if ec != 0 {
			return interp.NewExitStatus(ec)
		}
	}

	defExec := interp.DefaultExecHandler(2 * time.Second)
	return defExec(ctx, args)
}
