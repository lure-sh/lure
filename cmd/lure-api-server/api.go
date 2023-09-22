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

package main

import (
	"context"
	"io"

	"github.com/twitchtv/twirp"
	"go.elara.ws/lure/cmd/lure-api-server/internal/api"
	"go.elara.ws/lure/internal/log"
	"go.elara.ws/lure/pkg/search"
	"golang.org/x/text/language"
)

type lureWebAPI struct{}

func (l lureWebAPI) Search(ctx context.Context, req *api.SearchRequest) (*api.SearchResponse, error) {
	pkgs, err := search.Search(search.Options{
		Filter: search.Filter(req.FilterType),
		SortBy: search.SortBy(req.SortBy),
		Limit:  req.Limit,
		Query:  req.Query,
	})
	return &api.SearchResponse{Packages: searchPkgsToAPI(ctx, pkgs)}, err
}

func (l lureWebAPI) GetPkg(ctx context.Context, req *api.GetPackageRequest) (*api.Package, error) {
	pkg, err := search.GetPkg(req.Repository, req.Name)
	if err != nil {
		return nil, err
	}
	return searchPkgToAPI(ctx, pkg), nil
}

func (l lureWebAPI) GetBuildScript(ctx context.Context, req *api.GetBuildScriptRequest) (*api.GetBuildScriptResponse, error) {
	r, err := search.GetScript(req.Repository, req.Name)
	if err == search.ErrScriptNotFound {
		return nil, twirp.NewError(twirp.NotFound, err.Error())
	} else if err == search.ErrInvalidArgument {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	} else if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return &api.GetBuildScriptResponse{Script: string(data)}, nil
}

func searchPkgsToAPI(ctx context.Context, pkgs []search.Package) []*api.Package {
	out := make([]*api.Package, len(pkgs))
	for i, pkg := range pkgs {
		out[i] = searchPkgToAPI(ctx, pkg)
	}
	return out
}

func searchPkgToAPI(ctx context.Context, pkg search.Package) *api.Package {
	return &api.Package{
		Name:          pkg.Name,
		Repository:    pkg.Repository,
		Version:       pkg.Version,
		Release:       int64(pkg.Release),
		Epoch:         ptr(int64(pkg.Epoch)),
		Description:   performTranslation(ctx, pkg.Description),
		Homepage:      performTranslation(ctx, pkg.Homepage),
		Maintainer:    performTranslation(ctx, pkg.Maintainer),
		Architectures: pkg.Architectures,
		Licenses:      pkg.Licenses,
		Provides:      pkg.Provides,
		Conflicts:     pkg.Conflicts,
		Replaces:      pkg.Replaces,
		Depends:       mapToAPI(pkg.Depends),
		BuildDepends:  mapToAPI(pkg.BuildDepends),
	}
}

func ptr[T any](v T) *T {
	return &v
}

func mapToAPI(m map[string][]string) map[string]*api.StringList {
	out := make(map[string]*api.StringList, len(m))
	for override, list := range m {
		out[override] = &api.StringList{Entries: list}
	}
	return out
}

func performTranslation(ctx context.Context, v map[string]string) *string {
	alVal := ctx.Value(acceptLanguageKey{})
	langVal := ctx.Value(langParameterKey{})

	if alVal == nil && langVal == nil {
		val, ok := v[""]
		if !ok {
			return ptr("<unknown>")
		}
		return &val
	}

	al, _ := alVal.(string)
	lang, _ := langVal.(string)

	tags, _, err := language.ParseAcceptLanguage(al)
	if err != nil {
		log.Warn("Error parsing Accept-Language header").Err(err).Send()
	}

	var bases []string
	if lang != "" {
		langTag, err := language.Parse(lang)
		if err != nil {
			log.Warn("Error parsing lang parameter").Err(err).Send()
			bases = getLangBases(tags)
		} else {
			bases = getLangBases(append([]language.Tag{langTag}, tags...))
		}
	} else {
		bases = getLangBases(tags)
	}

	if len(bases) == 1 {
		bases = []string{"en", ""}
	}

	for _, name := range bases {
		val, ok := v[name]
		if !ok {
			continue
		}
		return &val
	}

	return ptr("<unknown>")
}

func getLangBases(langs []language.Tag) []string {
	out := make([]string, len(langs)+1)
	for i, lang := range langs {
		base, _ := lang.Base()
		out[i] = base.String()
	}
	out[len(out)-1] = ""
	return out
}
