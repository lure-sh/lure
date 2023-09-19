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

package decoder

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"go.elara.ws/lure/distro"
	"go.elara.ws/lure/internal/overrides"
	"golang.org/x/exp/slices"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var ErrNotPointerToStruct = errors.New("val must be a pointer to a struct")

type VarNotFoundError struct {
	name string
}

func (nfe VarNotFoundError) Error() string {
	return "required variable '" + nfe.name + "' could not be found"
}

type InvalidTypeError struct {
	name    string
	vartype string
	exptype string
}

func (ite InvalidTypeError) Error() string {
	return "variable '" + ite.name + "' is of type " + ite.vartype + ", but " + ite.exptype + " is expected"
}

// Decoder provides methods for decoding variable values
type Decoder struct {
	info   *distro.OSRelease
	Runner *interp.Runner
	// Enable distro overrides (true by default)
	Overrides bool
	// Enable using like distros for overrides
	LikeDistros bool
}

// New creates a new variable decoder
func New(info *distro.OSRelease, runner *interp.Runner) *Decoder {
	return &Decoder{info, runner, true, len(info.Like) > 0}
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
		DecodeHook: mapstructure.DecodeHookFuncValue(func(from, to reflect.Value) (interface{}, error) {
			if strings.Contains(to.Type().String(), "db.JSON") {
				valType := to.FieldByName("Val").Type()
				if !from.Type().AssignableTo(valType) {
					return nil, InvalidTypeError{name, from.Type().String(), valType.String()}
				}

				to.FieldByName("Val").Set(from)
				return to, nil
			}
			return from.Interface(), nil
		}),
		Result:  val,
		TagName: "sh",
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
		return ErrNotPointerToStruct
	} else {
		elemKind := reflect.TypeOf(val).Elem().Kind()
		if elemKind != reflect.Struct {
			return ErrNotPointerToStruct
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

type ScriptFunc func(ctx context.Context, opts ...interp.RunnerOption) error

// GetFunc returns a function corresponding to a bash function
// with the given name
func (d *Decoder) GetFunc(name string) (ScriptFunc, bool) {
	fn := d.getFunc(name)
	if fn == nil {
		return nil, false
	}

	return func(ctx context.Context, opts ...interp.RunnerOption) error {
		sub := d.Runner.Subshell()
		for _, opt := range opts {
			opt(sub)
		}
		return sub.Run(ctx, fn)
	}, true
}

func (d *Decoder) getFunc(name string) *syntax.Stmt {
	names, err := overrides.Resolve(d.info, overrides.DefaultOpts.WithName(name))
	if err != nil {
		return nil
	}

	for _, fnName := range names {
		fn, ok := d.Runner.Funcs[fnName]
		if ok {
			return fn
		}
	}
	return nil
}

// getVar gets a variable based on its name, taking into account
// override variables and nameref variables.
func (d *Decoder) getVar(name string) *expand.Variable {
	names, err := overrides.Resolve(d.info, overrides.DefaultOpts.WithName(name))
	if err != nil {
		return nil
	}

	for _, varName := range names {
		val, ok := d.Runner.Vars[varName]
		if ok {
			// Resolve nameref variables
			_, resolved := val.Resolve(expand.FuncEnviron(func(s string) string {
				if val, ok := d.Runner.Vars[s]; ok {
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
