package main

import (
	"github.com/jmoiron/sqlx"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
	_ "modernc.org/sqlite"
)

var gdb *sqlx.DB

func init() {
	var err error
	gdb, err = sqlx.Open("sqlite", config.DBPath)
	if err != nil {
		log.Fatal("Error opening database").Err(err).Send()
	}

	err = db.Init(gdb)
	if err != nil {
		log.Fatal("Error initializing database").Err(err).Send()
	}
}
