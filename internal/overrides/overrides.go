/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Elara Musayelyan
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

package overrides

import (
	"reflect"
	"strings"

	"go.elara.ws/lure/internal/cpu"
	"go.elara.ws/lure/pkg/db"
	"go.elara.ws/lure/pkg/distro"
	"golang.org/x/exp/slices"
	"golang.org/x/text/language"
)

type Opts struct {
	Name         string
	Overrides    bool
	LikeDistros  bool
	Languages    []string
	LanguageTags []language.Tag
}

var DefaultOpts = &Opts{
	Overrides:   true,
	LikeDistros: true,
	Languages:   []string{"en"},
}

// Resolve generates a slice of possible override names in the order that they should be checked
func Resolve(info *distro.OSRelease, opts *Opts) ([]string, error) {
	if opts == nil {
		opts = DefaultOpts
	}

	if !opts.Overrides {
		return []string{opts.Name}, nil
	}

	langs, err := parseLangs(opts.Languages, opts.LanguageTags)
	if err != nil {
		return nil, err
	}

	architectures, err := cpu.CompatibleArches(cpu.Arch())
	if err != nil {
		return nil, err
	}

	distros := []string{info.ID}
	if opts.LikeDistros {
		distros = append(distros, info.Like...)
	}

	var out []string
	for _, lang := range langs {
		for _, distro := range distros {
			for _, arch := range architectures {
				out = append(out, opts.Name+"_"+arch+"_"+distro+"_"+lang)
			}

			out = append(out, opts.Name+"_"+distro+"_"+lang)
		}

		for _, arch := range architectures {
			out = append(out, opts.Name+"_"+arch+"_"+lang)
		}

		out = append(out, opts.Name+"_"+lang)
	}

	for _, distro := range distros {
		for _, arch := range architectures {
			out = append(out, opts.Name+"_"+arch+"_"+distro)
		}

		out = append(out, opts.Name+"_"+distro)
	}

	for _, arch := range architectures {
		out = append(out, opts.Name+"_"+arch)
	}

	out = append(out, opts.Name)

	for index, item := range out {
		out[index] = strings.TrimPrefix(strings.ReplaceAll(item, "-", "_"), "_")
	}

	return out, nil
}

func (o *Opts) WithName(name string) *Opts {
	out := &Opts{}
	*out = *o

	out.Name = name
	return out
}

func (o *Opts) WithOverrides(v bool) *Opts {
	out := &Opts{}
	*out = *o

	out.Overrides = v
	return out
}

func (o *Opts) WithLikeDistros(v bool) *Opts {
	out := &Opts{}
	*out = *o

	out.LikeDistros = v
	return out
}

func (o *Opts) WithLanguages(langs []string) *Opts {
	out := &Opts{}
	*out = *o

	out.Languages = langs
	return out
}

func (o *Opts) WithLanguageTags(langs []string) *Opts {
	out := &Opts{}
	*out = *o

	out.Languages = langs
	return out
}

// ResolvedPackage is a LURE package after its overrides
// have been resolved
type ResolvedPackage struct {
	Name          string   `sh:"name"`
	Version       string   `sh:"version"`
	Release       int      `sh:"release"`
	Epoch         uint     `sh:"epoch"`
	Description   string   `db:"description"`
	Homepage      string   `db:"homepage"`
	Maintainer    string   `db:"maintainer"`
	Architectures []string `sh:"architectures"`
	Licenses      []string `sh:"license"`
	Provides      []string `sh:"provides"`
	Conflicts     []string `sh:"conflicts"`
	Replaces      []string `sh:"replaces"`
	Depends       []string `sh:"deps"`
	BuildDepends  []string `sh:"build_deps"`
	OptDepends    []string `sh:"opt_deps"`
}

func ResolvePackage(pkg *db.Package, overrides []string) *ResolvedPackage {
	out := &ResolvedPackage{}
	outVal := reflect.ValueOf(out).Elem()
	pkgVal := reflect.ValueOf(pkg).Elem()

	for i := 0; i < outVal.NumField(); i++ {
		fieldVal := outVal.Field(i)
		fieldType := fieldVal.Type()
		pkgFieldVal := pkgVal.FieldByName(outVal.Type().Field(i).Name)
		pkgFieldType := pkgFieldVal.Type()

		if strings.HasPrefix(pkgFieldType.String(), "db.JSON") {
			pkgFieldVal = pkgFieldVal.FieldByName("Val")
			pkgFieldType = pkgFieldVal.Type()
		}

		if pkgFieldType.AssignableTo(fieldType) {
			fieldVal.Set(pkgFieldVal)
			continue
		}

		if pkgFieldVal.Kind() == reflect.Map && pkgFieldType.Elem().AssignableTo(fieldType) {
			for _, override := range overrides {
				overrideVal := pkgFieldVal.MapIndex(reflect.ValueOf(override))
				if !overrideVal.IsValid() {
					continue
				}

				fieldVal.Set(overrideVal)
				break
			}
		}
	}

	return out
}

func parseLangs(langs []string, tags []language.Tag) ([]string, error) {
	out := make([]string, len(tags)+len(langs))
	for i, tag := range tags {
		base, _ := tag.Base()
		out[i] = base.String()
	}
	for i, lang := range langs {
		tag, err := language.Parse(lang)
		if err != nil {
			return nil, err
		}
		base, _ := tag.Base()
		out[len(tags)+i] = base.String()
	}
	slices.Sort(out)
	out = slices.Compact(out)
	return out, nil
}
