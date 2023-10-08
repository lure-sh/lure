package search

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lure.sh/lure/internal/config"
	"lure.sh/lure/internal/db"
)

// Filter represents search filters.
type Filter int

// Filters
const (
	FilterNone Filter = iota
	FilterInRepo
	FilterSupportsArch
)

// SoryBy represents a value that packages can be sorted by.
type SortBy int

// Sort values
const (
	SortByNone = iota
	SortByName
	SortByRepo
	SortByVersion
)

// Package represents a package from LURE's database
type Package struct {
	Name          string
	Version       string
	Release       int
	Epoch         uint
	Description   map[string]string
	Homepage      map[string]string
	Maintainer    map[string]string
	Architectures []string
	Licenses      []string
	Provides      []string
	Conflicts     []string
	Replaces      []string
	Depends       map[string][]string
	BuildDepends  map[string][]string
	OptDepends    map[string][]string
	Repository    string
}

func convertPkg(p db.Package) Package {
	return Package{
		Name:          p.Name,
		Version:       p.Version,
		Release:       p.Release,
		Epoch:         p.Epoch,
		Description:   p.Description.Val,
		Homepage:      p.Homepage.Val,
		Maintainer:    p.Maintainer.Val,
		Architectures: p.Architectures.Val,
		Licenses:      p.Licenses.Val,
		Provides:      p.Provides.Val,
		Conflicts:     p.Conflicts.Val,
		Replaces:      p.Replaces.Val,
		Depends:       p.Depends.Val,
		OptDepends:    p.OptDepends.Val,
		Repository:    p.Repository,
	}
}

// Options contains the options for a search.
type Options struct {
	Filter      Filter
	FilterValue string
	SortBy      SortBy
	Limit       int64
	Query       string
}

// Search searches for packages in the database based on the given options.
func Search(ctx context.Context, opts Options) ([]Package, error) {
	query := "(name LIKE ? OR description LIKE ? OR json_array_contains(provides, ?))"
	args := []any{"%" + opts.Query + "%", "%" + opts.Query + "%", opts.Query}

	if opts.Filter != FilterNone {
		switch opts.Filter {
		case FilterInRepo:
			query += " AND repository = ?"
		case FilterSupportsArch:
			query += " AND json_array_contains(architectures, ?)"
		}
		args = append(args, opts.FilterValue)
	}

	if opts.SortBy != SortByNone {
		switch opts.SortBy {
		case SortByName:
			query += " ORDER BY name"
		case SortByRepo:
			query += " ORDER BY repository"
		case SortByVersion:
			query += " ORDER BY version"
		}
	}

	if opts.Limit != 0 {
		query += " LIMIT " + strconv.FormatInt(opts.Limit, 10)
	}

	result, err := db.GetPkgs(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var out []Package
	for result.Next() {
		pkg := db.Package{}
		err = result.StructScan(&pkg)
		if err != nil {
			return nil, err
		}
		out = append(out, convertPkg(pkg))
	}

	return out, err
}

// GetPkg gets a single package from the database and returns it.
func GetPkg(ctx context.Context, repo, name string) (Package, error) {
	pkg, err := db.GetPkg(ctx, "name = ? AND repository = ?", name, repo)
	return convertPkg(*pkg), err
}

var (
	// ErrInvalidArgument is an error returned by GetScript when one of its arguments
	// contain invalid characters
	ErrInvalidArgument = errors.New("name and repository must not contain . or /")

	// ErrScriptNotFound is returned by GetScript if it can't find the script requested
	// by the user.
	ErrScriptNotFound = errors.New("requested script not found")
)

// GetScript returns a reader containing the build script for a given package.
func GetScript(ctx context.Context, repo, name string) (io.ReadCloser, error) {
	if strings.Contains(name, "./") || strings.ContainsAny(repo, "./") {
		return nil, ErrInvalidArgument
	}

	scriptPath := filepath.Join(config.GetPaths(ctx).RepoDir, repo, name, "lure.sh")
	fl, err := os.Open(scriptPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrScriptNotFound
	} else if err != nil {
		return nil, err
	}

	return fl, nil
}
