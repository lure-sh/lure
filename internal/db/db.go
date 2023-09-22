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

package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.elara.ws/lure/internal/log"
	"go.elara.ws/lure/internal/config"
	"golang.org/x/exp/slices"
	"modernc.org/sqlite"
)

// CurrentVersion is the current version of the database.
// The database is reset if its version doesn't match this.
const CurrentVersion = 2

func init() {
	sqlite.MustRegisterScalarFunction("json_array_contains", 2, jsonArrayContains)
}

// Package is a LURE package's database representation
type Package struct {
	Name          string                    `sh:"name,required" db:"name"`
	Version       string                    `sh:"version,required" db:"version"`
	Release       int                       `sh:"release,required" db:"release"`
	Epoch         uint                      `sh:"epoch" db:"epoch"`
	Description   JSON[map[string]string]   `db:"description"`
	Homepage      JSON[map[string]string]   `db:"homepage"`
	Maintainer    JSON[map[string]string]   `db:"maintainer"`
	Architectures JSON[[]string]            `sh:"architectures" db:"architectures"`
	Licenses      JSON[[]string]            `sh:"license" db:"licenses"`
	Provides      JSON[[]string]            `sh:"provides" db:"provides"`
	Conflicts     JSON[[]string]            `sh:"conflicts" db:"conflicts"`
	Replaces      JSON[[]string]            `sh:"replaces" db:"replaces"`
	Depends       JSON[map[string][]string] `db:"depends"`
	BuildDepends  JSON[map[string][]string] `db:"builddepends"`
	OptDepends    JSON[map[string][]string] `db:"optdepends"`
	Repository    string                    `db:"repository"`
}

type version struct {
	Version int `db:"version"`
}

var (
	conn   *sqlx.DB
	closed = true
)

// DB returns the LURE database.
// The first time it's called, it opens the SQLite database file.
// Subsequent calls return the same connection.
func DB() *sqlx.DB {
	if conn != nil && !closed {
		return conn
	}
	db, err := open(config.GetPaths().DBPath)
	if err != nil {
		log.Fatal("Error opening database").Err(err).Send()
	}
	conn = db
	return conn
}

func open(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	conn = db
	closed = false

	err = initDB(dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Close closes the database
func Close() error {
	closed = true
	if conn != nil {
		return conn.Close()
	} else {
		return nil
	}
}

// initDB initializes the database
func initDB(dsn string) error {
	conn = conn.Unsafe()
	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS pkgs (
			name          TEXT NOT NULL,
			repository    TEXT NOT NULL,
			version       TEXT NOT NULL,
			release       INT  NOT NULL,
			epoch         INT,
			description   TEXT CHECK(description = 'null' OR (JSON_VALID(description) AND JSON_TYPE(description) = 'object')),
			homepage      TEXT CHECK(homepage = 'null' OR (JSON_VALID(homepage) AND JSON_TYPE(homepage) = 'object')),
			maintainer    TEXT CHECK(maintainer = 'null' OR (JSON_VALID(maintainer) AND JSON_TYPE(maintainer) = 'object')),
			architectures TEXT CHECK(architectures = 'null' OR (JSON_VALID(architectures) AND JSON_TYPE(architectures) = 'array')),
			licenses      TEXT CHECK(licenses = 'null' OR (JSON_VALID(licenses) AND JSON_TYPE(licenses) = 'array')),
			provides      TEXT CHECK(provides = 'null' OR (JSON_VALID(provides) AND JSON_TYPE(provides) = 'array')),
			conflicts     TEXT CHECK(conflicts = 'null' OR (JSON_VALID(conflicts) AND JSON_TYPE(conflicts) = 'array')),
			replaces      TEXT CHECK(replaces = 'null' OR (JSON_VALID(replaces) AND JSON_TYPE(replaces) = 'array')),
			depends       TEXT CHECK(depends = 'null' OR (JSON_VALID(depends) AND JSON_TYPE(depends) = 'object')),
			builddepends  TEXT CHECK(builddepends = 'null' OR (JSON_VALID(builddepends) AND JSON_TYPE(builddepends) = 'object')),
			optdepends    TEXT CHECK(optdepends = 'null' OR (JSON_VALID(optdepends) AND JSON_TYPE(optdepends) = 'object')),
			UNIQUE(name, repository)
		);

		CREATE TABLE IF NOT EXISTS lure_db_version (
			version INT NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	ver, ok := GetVersion()
	if ok && ver != CurrentVersion {
		log.Warn("Database version mismatch; resetting").Int("version", ver).Int("expected", CurrentVersion).Send()
		reset()
		return initDB(dsn)
	} else if !ok {
		log.Warn("Database version does not exist. Run lure fix if something isn't working.").Send()
		return addVersion(CurrentVersion)
	}

	return nil
}

// reset drops all the database tables
func reset() error {
	_, err := DB().Exec("DROP TABLE IF EXISTS pkgs;")
	if err != nil {
		return err
	}
	_, err = DB().Exec("DROP TABLE IF EXISTS lure_db_version;")
	return err
}

// IsEmpty returns true if the database has no packages in it, otherwise it returns false.
func IsEmpty() bool {
	var count int
	err := DB().Get(&count, "SELECT count(1) FROM pkgs;")
	if err != nil {
		return true
	}
	return count == 0
}

// GetVersion returns the database version and a boolean indicating
// whether the database contained a version number
func GetVersion() (int, bool) {
	var ver version
	err := DB().Get(&ver, "SELECT * FROM lure_db_version LIMIT 1;")
	if err != nil {
		return 0, false
	}
	return ver.Version, true
}

func addVersion(ver int) error {
	_, err := DB().Exec(`INSERT INTO lure_db_version(version) VALUES (?);`, ver)
	return err
}

// InsertPackage adds a package to the database
func InsertPackage(pkg Package) error {
	_, err := DB().NamedExec(`
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
			builddepends,
			optdepends
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
			:builddepends,
			:optdepends
		);
	`, pkg)
	return err
}

// GetPkgs returns a result containing packages that match the where conditions
func GetPkgs(where string, args ...any) (*sqlx.Rows, error) {
	stream, err := DB().Queryx("SELECT * FROM pkgs WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// GetPkg returns a single package that matches the where conditions
func GetPkg(where string, args ...any) (*Package, error) {
	out := &Package{}
	err := DB().Get(out, "SELECT * FROM pkgs WHERE "+where+" LIMIT 1", args...)
	return out, err
}

// DeletePkgs deletes all packages matching the where conditions
func DeletePkgs(where string, args ...any) error {
	_, err := DB().Exec("DELETE FROM pkgs WHERE "+where, args...)
	return err
}

// jsonArrayContains is an SQLite function that checks if a JSON array
// in the database contains a given value
func jsonArrayContains(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	value, ok := args[0].(string)
	if !ok {
		return nil, errors.New("both arguments to json_array_contains must be strings")
	}

	item, ok := args[1].(string)
	if !ok {
		return nil, errors.New("both arguments to json_array_contains must be strings")
	}

	var array []string
	err := json.Unmarshal([]byte(value), &array)
	if err != nil {
		return nil, err
	}

	return slices.Contains(array, item), nil
}

// JSON represents a JSON value in the database
type JSON[T any] struct {
	Val T
}

// NewJSON creates a new database JSON value
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
