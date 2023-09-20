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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/twitchtv/twirp"
	"go.elara.ws/lure/internal/api"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/internal/log"
	"golang.org/x/text/language"
)

type lureWebAPI struct{}

func (l lureWebAPI) Search(ctx context.Context, req *api.SearchRequest) (*api.SearchResponse, error) {
	query := "(name LIKE ? OR description LIKE ? OR json_array_contains(provides, ?))"
	args := []any{"%" + req.Query + "%", "%" + req.Query + "%", req.Query}

	if req.FilterValue != nil && req.FilterType != api.FILTER_TYPE_NO_FILTER {
		switch req.FilterType {
		case api.FILTER_TYPE_IN_REPOSITORY:
			query += " AND repository = ?"
		case api.FILTER_TYPE_SUPPORTS_ARCH:
			query += " AND json_array_contains(architectures, ?)"
		}
		args = append(args, *req.FilterValue)
	}

	if req.SortBy != api.SORT_BY_UNSORTED {
		switch req.SortBy {
		case api.SORT_BY_NAME:
			query += " ORDER BY name"
		case api.SORT_BY_REPOSITORY:
			query += " ORDER BY repository"
		case api.SORT_BY_VERSION:
			query += " ORDER BY version"
		}
	}

	if req.Limit != 0 {
		query += " LIMIT " + strconv.FormatInt(req.Limit, 10)
	}

	result, err := db.GetPkgs(query, args...)
	if err != nil {
		return nil, err
	}

	out := &api.SearchResponse{}
	for result.Next() {
		pkg := &db.Package{}
		err = result.StructScan(pkg)
		if err != nil {
			return nil, err
		}
		out.Packages = append(out.Packages, dbPkgToAPI(ctx, pkg))
	}

	return out, err
}

func (l lureWebAPI) GetPkg(ctx context.Context, req *api.GetPackageRequest) (*api.Package, error) {
	pkg, err := db.GetPkg("name = ? AND repository = ?", req.Name, req.Repository)
	if err != nil {
		return nil, err
	}
	return dbPkgToAPI(ctx, pkg), nil
}

func (l lureWebAPI) GetBuildScript(ctx context.Context, req *api.GetBuildScriptRequest) (*api.GetBuildScriptResponse, error) {
	if strings.ContainsAny(req.Name, "./") || strings.ContainsAny(req.Repository, "./") {
		return nil, twirp.NewError(twirp.InvalidArgument, "name and repository must not contain . or /")
	}

	scriptPath := filepath.Join(config.GetPaths().RepoDir, req.Repository, req.Name, "lure.sh")
	_, err := os.Stat(scriptPath)
	if os.IsNotExist(err) {
		return nil, twirp.NewError(twirp.NotFound, "requested package not found")
	} else if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}

	return &api.GetBuildScriptResponse{Script: string(data)}, nil
}

func dbPkgToAPI(ctx context.Context, pkg *db.Package) *api.Package {
	return &api.Package{
		Name:          pkg.Name,
		Repository:    pkg.Repository,
		Version:       pkg.Version,
		Release:       int64(pkg.Release),
		Epoch:         ptr(int64(pkg.Epoch)),
		Description:   performTranslation(ctx, pkg.Description.Val),
		Homepage:      performTranslation(ctx, pkg.Homepage.Val),
		Maintainer:    performTranslation(ctx, pkg.Maintainer.Val),
		Architectures: pkg.Architectures.Val,
		Licenses:      pkg.Licenses.Val,
		Provides:      pkg.Provides.Val,
		Conflicts:     pkg.Conflicts.Val,
		Replaces:      pkg.Replaces.Val,
		Depends:       dbMapToAPI(pkg.Depends.Val),
		BuildDepends:  dbMapToAPI(pkg.BuildDepends.Val),
	}
}

func ptr[T any](v T) *T {
	return &v
}

func dbMapToAPI(m map[string][]string) map[string]*api.StringList {
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
