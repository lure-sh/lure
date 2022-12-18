package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
	"github.com/genjidb/genji/types"
	"go.arsenm.dev/lure/internal/api"
	"go.arsenm.dev/lure/internal/db"
)

type lureWebAPI struct {
	db *genji.DB
}

func (l lureWebAPI) Search(ctx context.Context, req *api.SearchRequest) (*api.SearchResponse, error) {
	query := "(name LIKE ? OR description LIKE ? OR ? IN provides)"
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

	doc, err := db.GetPkgs(l.db, query, args...)
	if err != nil {
		return nil, err
	}

	fmt.Println(query, args)
	out := &api.SearchResponse{}
	err = doc.Iterate(func(d types.Document) error {
		pkg := &db.Package{}
		err = document.ScanDocument(d, pkg)
		if err != nil {
			return err
		}
		out.Packages = append(out.Packages, dbPkgToAPI(pkg))
		return nil
	})
	return out, err
}

func (l lureWebAPI) GetPkg(ctx context.Context, req *api.GetPackageRequest) (*api.Package, error) {
	pkg, err := db.GetPkg(l.db, "name = ? AND repository = ?", req.Name, req.Repository)
	if err != nil {
		return nil, err
	}
	return dbPkgToAPI(pkg), nil
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
		Architectures: pkg.Architectures,
		Licenses:      pkg.Licenses,
		Provides:      pkg.Provides,
		Conflicts:     pkg.Conflicts,
		Replaces:      pkg.Replaces,
		Depends:       dbMapToAPI(pkg.Depends),
		BuildDepends:  dbMapToAPI(pkg.BuildDepends),
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
