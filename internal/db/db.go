package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"golang.org/x/exp/slices"
	"modernc.org/sqlite"
)

const CurrentVersion = 1

func init() {
	sqlite.MustRegisterScalarFunction("json_array_contains", 2, JsonArrayContains)
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
	Repository    string                    `db:"repository"`
}

type version struct {
	Version int `db:"version"`
}

func Open(dsn string) (*sqlx.DB, error) {
	if dsn != ":memory:" {
		fi, err := os.Stat(config.DBPath)
		if err == nil {
			// TODO: This should be removed by the first stable release.
			if fi.IsDir() {
				log.Warn("Your database is using the old database engine; rebuilding").Send()

				err = os.RemoveAll(config.DBPath)
				if err != nil {
					log.Fatal("Error removing old database").Err(err).Send()
				}
				config.DBPresent = false
			}
		}
	}

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		log.Fatal("Error opening database").Err(err).Send()
	}

	err = Init(db, dsn)
	if err != nil {
		log.Fatal("Error initializing database").Err(err).Send()
	}

	return db, nil
}

// Init initializes the database
func Init(db *sqlx.DB, dsn string) error {
	*db = *db.Unsafe()
	_, err := db.Exec(`
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
			UNIQUE(name, repository)
		);

		CREATE TABLE IF NOT EXISTS lure_db_version (
			version INT NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	ver, ok := GetVersion(db)
	if !ok {
		return addVersion(db, CurrentVersion)
	}

	if ver != CurrentVersion {
		log.Warn("Database version mismatch; rebuilding").Int("version", ver).Int("expected", CurrentVersion).Send()

		db.Close()
		err = os.Remove(config.DBPath)
		if err != nil {
			return err
		}
		config.DBPresent = false

		tdb, err := Open(dsn)
		if err != nil {
			return err
		}
		*db = *tdb
	}

	return nil
}

func GetVersion(db *sqlx.DB) (int, bool) {
	var ver version
	err := db.Get(&ver, "SELECT * FROM lure_db_version LIMIT 1;")
	if err != nil {
		return 0, false
	}
	return ver.Version, true
}

func addVersion(db *sqlx.DB, ver int) error {
	_, err := db.Exec(`INSERT INTO lure_db_version(version) VALUES (?);`, ver)
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
	stream, err := db.Queryx("SELECT * FROM pkgs WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// GetPkg returns a single package that match the where conditions
func GetPkg(db *sqlx.DB, where string, args ...any) (*Package, error) {
	out := &Package{}
	err := db.Get(out, "SELECT * FROM pkgs WHERE "+where+" LIMIT 1", args...)
	return out, err
}

// DeletePkgs deletes all packages matching the where conditions
func DeletePkgs(db *sqlx.DB, where string, args ...any) error {
	_, err := db.Exec("DELETE FROM pkgs WHERE "+where, args...)
	return err
}

func JsonArrayContains(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
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
