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
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/exp/slices"
	"mvdan.cc/sh/v3/interp"
)

func RestrictedReadDir(allowedPrefixes ...string) interp.ReadDirHandlerFunc {
	return func(ctx context.Context, s string) ([]os.FileInfo, error) {
		for _, allowedPrefix := range allowedPrefixes {
			if strings.HasPrefix(s, allowedPrefix) {
				return interp.DefaultReadDirHandler()(ctx, s)
			}
		}

		return nil, os.ErrNotExist
	}
}

func RestrictedStat(allowedPrefixes ...string) interp.StatHandlerFunc {
	return func(ctx context.Context, s string, b bool) (os.FileInfo, error) {
		for _, allowedPrefix := range allowedPrefixes {
			if strings.HasPrefix(s, allowedPrefix) {
				return interp.DefaultStatHandler()(ctx, s, b)
			}
		}

		return nil, os.ErrNotExist
	}
}

func RestrictedOpen(allowedPrefixes ...string) interp.OpenHandlerFunc {
	return func(ctx context.Context, s string, i int, fm os.FileMode) (io.ReadWriteCloser, error) {
		for _, allowedPrefix := range allowedPrefixes {
			if strings.HasPrefix(s, allowedPrefix) {
				return interp.DefaultOpenHandler()(ctx, s, i, fm)
			}
		}

		return NopRWC{}, nil
	}
}

func RestrictedExec(allowedCmds ...string) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if slices.Contains(allowedCmds, args[0]) {
			return interp.DefaultExecHandler(2*time.Second)(ctx, args)
		}

		return nil
	}
}
