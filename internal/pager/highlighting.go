/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Arsen Musayelyan
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

package pager

import (
	"bytes"
	"io"

	"github.com/alecthomas/chroma/v2/quick"
)

func SyntaxHighlightBash(r io.Reader, style string) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	w := &bytes.Buffer{}
	err = quick.Highlight(w, string(data), "bash", "terminal", style)
	return w.String(), err
}
