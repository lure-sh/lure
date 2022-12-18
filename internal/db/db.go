package db

import (
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/types"
)

// Package is a LURE package's database representation
type Package struct {
	Name          string   `sh:"name,required"`
	Version       string   `sh:"version,required"`
	Release       int      `sh:"release,required"`
	Epoch         uint     `sh:"epoch"`
	Description   string   `sh:"desc"`
	Homepage      string   `sh:"homepage"`
	Maintainer    string   `sh:"maintainer"`
	Architectures []string `sh:"architectures"`
	Licenses      []string `sh:"license"`
	Provides      []string `sh:"provides"`
	Conflicts     []string `sh:"conflicts"`
	Replaces      []string `sh:"replaces"`
	Depends       map[string][]string
	BuildDepends  map[string][]string
	Repository    string
}

// Package is a LURE Web comment's database representation
type Comment struct {
	CommentID   int64
	PackageName string
	PackageRepo string
	TimeCreated int64
	Contents    string
}

// Init initializes the database
func Init(db *genji.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS pkgs (
			name          TEXT NOT NULL,
			repository    TEXT NOT NULL,
			version       TEXT NOT NULL,
			release       INT  NOT NULL,
			epoch         INT,
			description   TEXT,
			homepage      TEXT,
			maintainer    TEXT,
			architectures ARRAY,
			licenses      ARRAY,
			provides      ARRAY,
			conflicts     ARRAY,
			replaces      ARRAY,
			depends       (...),
			builddepends  (...),
			UNIQUE(name, repository)
		);

		CREATE TABLE IF NOT EXISTS comments (
			comment_id   INT  PRIMARY KEY,
			package_name TEXT NOT NULL,
			package_repo TEXT NOT NULL,
			time_created INT  NOT NULL,
			contents     TEXT NOT NULL,
			UNIQUE(comment_id),
			UNIQUE(package_name, package_repo)
		);
	`)
}

// InsertPackage adds a package to the database
func InsertPackage(db *genji.DB, pkg Package) error {
	return db.Exec("INSERT INTO pkgs VALUES ? ON CONFLICT DO REPLACE;", pkg)
}

// GetPkgs returns a result containing packages that match the where conditions
func GetPkgs(db *genji.DB, where string, args ...any) (*genji.Result, error) {
	stream, err := db.Query("SELECT * FROM pkgs WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// DeletePkgs deletes all packages matching the where conditions
func DeletePkgs(db *genji.DB, where string, args ...any) error {
	return db.Exec("DELETE FROM pkgs WHERE "+where, args...)
}

func InsertComment(db *genji.DB, c Comment) error {
	return db.Exec("INSERT INTO comments VALUES ? ON CONFLICT DO REPLACE;", c)
}

func CountComments(db *genji.DB) (int64, error) {
	doc, err := db.QueryDocument("SELECT count(*) FROM comments;")
	if err != nil {
		return 0, err
	}
	val, err := doc.GetByField("COUNT(*)")
	if err != nil {
		return 0, err
	}
	return val.V().(int64), nil
}

// GetComments returns a result containing comments that match the where conditions
func GetComments(db *genji.DB, where string, args ...any) (*genji.Result, error) {
	stream, err := db.Query("SELECT * FROM comments WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// GetComment returns a comment that matches the where conditions
func GetComment(db *genji.DB, where string, args ...any) (types.Document, error) {
	doc, err := db.QueryDocument("SELECT * FROM comments WHERE "+where+" LIMIT 1", args...)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// DeleteComments deletes all comments matching the where conditions
func DeleteComments(db *genji.DB, where string, args ...any) error {
	return db.Exec("DELETE FROM comments WHERE "+where, args...)
}
