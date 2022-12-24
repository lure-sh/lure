package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// Package is a LURE package's database representation
type Package struct {
	Name          string                    `sh:"name,required" db:"name"`
	Version       string                    `sh:"version,required" db:"version"`
	Release       int                       `sh:"release,required" db:"release"`
	Epoch         uint                      `sh:"epoch" db:"epoch"`
	Description   string                    `sh:"desc" db:"description"`
	Homepage      string                    `sh:"homepage" db:"homepage"`
	Maintainer    string                    `sh:"maintainer" db:"maintainer"`
	Architectures JSON[[]string]            `sh:"architectures" db:"architectures"`
	Licenses      JSON[[]string]            `sh:"license" db:"licenses"`
	Provides      JSON[[]string]            `sh:"provides" db:"provides"`
	Conflicts     JSON[[]string]            `sh:"conflicts" db:"conflicts"`
	Replaces      JSON[[]string]            `sh:"replaces" db:"replaces"`
	Depends       JSON[map[string][]string] `db:"depends"`
	BuildDepends  JSON[map[string][]string] `db:"builddepends"`
	Repository    string                    `db:"repository"`
}

// Init initializes the database
func Init(db *sqlx.DB) error {
	*db = *db.Unsafe()
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS pkgs (
			name          TEXT NOT NULL,
			repository    TEXT NOT NULL,
			version       TEXT NOT NULL,
			release       INT  NOT NULL,
			epoch         INT,
			description   TEXT,
			homepage      TEXT,
			maintainer    TEXT,
			architectures TEXT CHECK(architectures = 'null' OR (JSON_VALID(architectures) AND JSON_TYPE(architectures) = 'array')),
			licenses      TEXT CHECK(licenses = 'null' OR (JSON_VALID(licenses) AND JSON_TYPE(licenses) = 'array')),
			provides      TEXT CHECK(provides = 'null' OR (JSON_VALID(provides) AND JSON_TYPE(provides) = 'array')),
			conflicts     TEXT CHECK(conflicts = 'null' OR (JSON_VALID(conflicts) AND JSON_TYPE(conflicts) = 'array')),
			replaces      TEXT CHECK(replaces = 'null' OR (JSON_VALID(replaces) AND JSON_TYPE(replaces) = 'array')),
			depends       TEXT CHECK(depends = 'null' OR (JSON_VALID(depends) AND JSON_TYPE(depends) = 'object')),
			builddepends  TEXT CHECK(builddepends = 'null' OR (JSON_VALID(builddepends) AND JSON_TYPE(builddepends) = 'object')),
			UNIQUE(name, repository)
		);
	`)
	return err
}

// InsertPackage adds a package to the database
func InsertPackage(db *sqlx.DB, pkg Package) error {
	_, err := db.NamedExec(`
		INSERT OR REPLACE INTO pkgs (
			name,
			repository,
			version,
			release,
			epoch,
			description,
			homepage,
			maintainer,
			architectures,
			licenses,
			provides,
			conflicts,
			replaces,
			depends,
			builddepends
		) VALUES (
			:name,
			:repository,
			:version,
			:release,
			:epoch,
			:description,
			:homepage,
			:maintainer,
			:architectures,
			:licenses,
			:provides,
			:conflicts,
			:replaces,
			:depends,
			:builddepends
		);
	`, pkg)
	return err
}

// GetPkgs returns a result containing packages that match the where conditions
func GetPkgs(db *sqlx.DB, where string, args ...any) (*sqlx.Rows, error) {
	stream, err := db.Queryx("SELECT DISTINCT * FROM pkgs, json_each(pkgs.provides) AS provides WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// GetPkg returns a single package that match the where conditions
func GetPkg(db *sqlx.DB, where string, args ...any) (*Package, error) {
	out := &Package{}
	err := db.Get(out, "SELECT DISTINCT * FROM pkgs, json_each(pkgs.provides) AS provides WHERE "+where+"LIMIT 1", args...)
	return out, err
}

// DeletePkgs deletes all packages matching the where conditions
func DeletePkgs(db *sqlx.DB, where string, args ...any) error {
	_, err := db.Exec("DELETE FROM pkgs WHERE "+where, args...)
	return err
}

type JSON[T any] struct {
	Val T
}

func NewJSON[T any](v T) JSON[T] {
	return JSON[T]{Val: v}
}

func (s *JSON[T]) Scan(val any) error {
	if val == nil {
		return nil
	}

	switch val := val.(type) {
	case string:
		err := json.Unmarshal([]byte(val), &s.Val)
		if err != nil {
			return err
		}
	case sql.NullString:
		if val.Valid {
			err := json.Unmarshal([]byte(val.String), &s.Val)
			if err != nil {
				return err
			}
		}
	default:
		return errors.New("sqlite json types must be strings")
	}

	return nil
}

func (s JSON[T]) Value() (driver.Value, error) {
	data, err := json.Marshal(s.Val)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (s JSON[T]) MarshalYAML() (any, error) {
	return s.Val, nil
}

func (s JSON[T]) String() string {
	return fmt.Sprint(s.Val)
}

func (s JSON[T]) GoString() string {
	return fmt.Sprintf("%#v", s.Val)
}
