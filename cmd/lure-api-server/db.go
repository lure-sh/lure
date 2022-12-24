package main

import (
	"os"

	"github.com/jmoiron/sqlx"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
	"modernc.org/sqlite"
)

var gdb *sqlx.DB

func init() {
	fi, err := os.Stat(config.DBPath)
	if err != nil {
		log.Fatal("Cannot stat database path").Err(err).Send()
	}

	// TODO: This should be removed by the first stable release.
	if fi.IsDir() {
		log.Fatal("Your package cache database is using the old database engine. Please remove ~/.cache/lure and then run `lure ref`.").Send()
	}

	sqlite.MustRegisterScalarFunction("json_array_contains", 2, db.JsonArrayContains)

	gdb, err = sqlx.Open("sqlite", config.DBPath)
	if err != nil {
		log.Fatal("Error opening database").Err(err).Send()
	}

	err = db.Init(gdb)
	if err != nil {
		log.Fatal("Error initializing database").Err(err).Send()
	}
}
