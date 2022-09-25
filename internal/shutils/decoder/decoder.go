package decoder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"

	"github.com/mitchellh/mapstructure"
	"go.arsenm.dev/lure/distro"
	"golang.org/x/exp/slices"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var ErrInvalidType = errors.New("val must be a pointer to a struct")

type NotFoundError struct {
	stype string
	name  string
}

func (nfe NotFoundError) Error() string {
	return "required " + nfe.stype + " '" + nfe.name + "' could not be found"
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
		return NotFoundError{"variable", name}
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
		if _, ok := err.(NotFoundError); ok && !required {
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

// WriteFunc writes the contents of a bash function to w.
func (d *Decoder) WriteFunc(name string, w io.Writer) error {
	fn := d.getFunc(name)
	if fn == nil {
		return NotFoundError{"function", name}
	}

	printer := syntax.NewPrinter()

	// Print individual statements instead of the entire block
	block := fn.Cmd.(*syntax.Block)
	for _, stmt := range block.Stmts {
		err := printer.Print(w, stmt)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
	}

	return nil
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

	return []string{
		fmt.Sprintf("%s_%s_%s", name, runtime.GOARCH, d.info.ID),
		fmt.Sprintf("%s_%s", name, d.info.ID),
		fmt.Sprintf("%s_%s", name, runtime.GOARCH),
		name,
	}
}
