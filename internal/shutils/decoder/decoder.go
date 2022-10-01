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

package decoder

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/mitchellh/mapstructure"
	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/cpu"
	"golang.org/x/exp/slices"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var ErrInvalidType = errors.New("val must be a pointer to a struct")

type VarNotFoundError struct {
	name string
}

func (nfe VarNotFoundError) Error() string {
	return "required variable '" + nfe.name + "' could not be found"
}

// Decoder provides methods for decoding variable values
type Decoder struct {
	info      *distro.OSRelease
	runner    *interp.Runner
	Overrides bool
}

// New creates a new variable decoder
func New(info *distro.OSRelease, runner *interp.Runner) *Decoder {
	return &Decoder{info, runner, true}
}

// DecodeVar decodes a variable to val using reflection.
// Structs should use the "sh" struct tag.
func (d *Decoder) DecodeVar(name string, val any) error {
	variable := d.getVar(name)
	if variable == nil {
		return VarNotFoundError{name}
	}

	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           val,
		TagName:          "sh",
	})
	if err != nil {
		return err
	}

	switch variable.Kind {
	case expand.Indexed:
		return dec.Decode(variable.List)
	case expand.Associative:
		return dec.Decode(variable.Map)
	default:
		return dec.Decode(variable.Str)
	}
}

// DecodeVars decodes all variables to val using reflection.
// Structs should use the "sh" struct tag.
func (d *Decoder) DecodeVars(val any) error {
	valKind := reflect.TypeOf(val).Kind()
	if valKind != reflect.Pointer {
		return ErrInvalidType
	} else {
		elemKind := reflect.TypeOf(val).Elem().Kind()
		if elemKind != reflect.Struct {
			return ErrInvalidType
		}
	}

	rVal := reflect.ValueOf(val).Elem()

	for i := 0; i < rVal.NumField(); i++ {
		field := rVal.Field(i)
		fieldType := rVal.Type().Field(i)

		if !fieldType.IsExported() {
			continue
		}

		name := fieldType.Name
		tag := fieldType.Tag.Get("sh")
		required := false
		if tag != "" {
			if strings.Contains(tag, ",") {
				splitTag := strings.Split(tag, ",")
				name = splitTag[0]

				if len(splitTag) > 1 {
					if slices.Contains(splitTag, "required") {
						required = true
					}
				}
			} else {
				name = tag
			}
		}

		newVal := reflect.New(field.Type())
		err := d.DecodeVar(name, newVal.Interface())
		if _, ok := err.(VarNotFoundError); ok && !required {
			continue
		} else if err != nil {
			return err
		}

		field.Set(newVal.Elem())
	}

	return nil
}

type ScriptFunc func(ctx context.Context, sir string, args ...string) error

// GetFunc returns a function corresponding to a bash function
// with the given name
func (d *Decoder) GetFunc(name string) (ScriptFunc, bool) {
	fn := d.getFunc(name)
	if fn == nil {
		return nil, false
	}

	return func(ctx context.Context, dir string, args ...string) error {
		sub := d.runner.Subshell()
		interp.Params(args...)(sub)
		interp.Dir(dir)(sub)
		return sub.Run(ctx, fn)
	}, true
}

func (d *Decoder) getFunc(name string) *syntax.Stmt {
	names := d.genPossibleNames(name)
	for _, fnName := range names {
		fn, ok := d.runner.Funcs[fnName]
		if ok {
			return fn
		}
	}
	return nil
}

// getVar gets a variable based on its name, taking into account
// override variables and nameref variables.
func (d *Decoder) getVar(name string) *expand.Variable {
	names := d.genPossibleNames(name)
	for _, varName := range names {
		val, ok := d.runner.Vars[varName]
		if ok {
			// Resolve nameref variables
			_, resolved := val.Resolve(expand.FuncEnviron(func(s string) string {
				if val, ok := d.runner.Vars[s]; ok {
					return val.String()
				}
				return ""
			}))
			val = resolved

			return &val
		}
	}
	return nil
}

// genPossibleNames generates a slice of the possible names that
// could be used in the order that they should be checked
func (d *Decoder) genPossibleNames(name string) []string {
	if !d.Overrides {
		return []string{name}
	}

	architectures := []string{runtime.GOARCH}

	if runtime.GOARCH == "arm" {
		// More specific goes first
		architectures[0] = cpu.ARMVariant()
		architectures = append(architectures, "arm")
	}

	distros := []string{d.info.ID}
	distros = append(distros, d.info.Like...)

	var out []string
	for _, arch := range architectures {
		for _, distro := range distros {
			out = append(
				out,
				fmt.Sprintf("%s_%s_%s", name, arch, distro),
				fmt.Sprintf("%s_%s", name, distro),
			)
		}
		out = append(out, fmt.Sprintf("%s_%s", name, arch))
	}
	out = append(out, name)

	for index, item := range out {
		out[index] = strings.ReplaceAll(item, "-", "_")
	}

	return out
}
