package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/twitchtv/twirp"
	"go.arsenm.dev/lure/internal/api"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
)

type lureWebAPI struct {
	db *sqlx.DB
}

func (l lureWebAPI) Search(ctx context.Context, req *api.SearchRequest) (*api.SearchResponse, error) {
	query := "(name LIKE ? OR description LIKE ? OR provides.value = ?)"
	args := []any{"%" + req.Query + "%", "%" + req.Query + "%", req.Query}

	if req.FilterValue != nil && req.FilterType != api.FILTER_TYPE_NO_FILTER {
		switch req.FilterType {
		case api.FILTER_TYPE_IN_REPOSITORY:
			query += " AND repository = ?"
		case api.FILTER_TYPE_SUPPORTS_ARCH:
			query += " AND ? IN architectures"
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

	result, err := db.GetPkgs(l.db, query, args...)
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
		out.Packages = append(out.Packages, dbPkgToAPI(pkg))
	}

	return out, err
}

func (l lureWebAPI) GetPkg(ctx context.Context, req *api.GetPackageRequest) (*api.Package, error) {
	pkg, err := db.GetPkg(l.db, "name = ? AND repository = ?", req.Name, req.Repository)
	if err != nil {
		return nil, err
	}
	return dbPkgToAPI(pkg), nil
}

func (l lureWebAPI) GetBuildScript(ctx context.Context, req *api.GetBuildScriptRequest) (*api.GetBuildScriptResponse, error) {
	if strings.ContainsAny(req.Name, "./") || strings.ContainsAny(req.Repository, "./") {
		return nil, twirp.NewError(twirp.InvalidArgument, "name and repository must not contain . or /")
	}

	scriptPath := filepath.Join(config.RepoDir, req.Repository, req.Name, "lure.sh")
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

func dbPkgToAPI(pkg *db.Package) *api.Package {
	return &api.Package{
		Name:          pkg.Name,
		Repository:    pkg.Repository,
		Version:       pkg.Version,
		Release:       int64(pkg.Release),
		Epoch:         ptr(int64(pkg.Epoch)),
		Description:   &pkg.Description,
		Homepage:      &pkg.Homepage,
		Maintainer:    &pkg.Maintainer,
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
