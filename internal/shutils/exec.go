/*
 * LURE - Linux User REpository
 * Copyright (C) 2022 Arsen Musayelyan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
