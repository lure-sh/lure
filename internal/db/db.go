package db

import (
	"github.com/genjidb/genji"
	"github.com/genjidb/genji/document"
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

// Init initializes the database
func Init(db *genji.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS pkgs (
			name       TEXT NOT NULL,
			repository TEXT NOT NULL,
			version    TEXT NOT NULL,
			release    INT  NOT NULL,
			epoch      INT,
			description TEXT,
			homepage TEXT,
			maintainer TEXT,
			architectures ARRAY,
			licenses ARRAY,
			provides ARRAY,
			conflicts ARRAY,
			replaces ARRAY,
			depends (...),
			builddepends (...),
			UNIQUE(name, repository)
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

// GetPkg returns a single package matching the where conditions
func GetPkg(db *genji.DB, where string, args ...any) (*Package, error) {
	doc, err := db.QueryDocument("SELECT * FROM pkgs WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	out := &Package{}
	err = document.StructScan(doc, out)
	return out, err
}

// DeletePkgs deletes all packages matching the where conditions
func DeletePkgs(db *genji.DB, where string, args ...any) error {
	return db.Exec("DELETE * FROM pkgs WHERE "+where, args...)
}
