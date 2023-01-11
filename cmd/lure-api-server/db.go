package main

import (
	"github.com/jmoiron/sqlx"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
)

var gdb *sqlx.DB

func init() {
	var err error
	gdb, err = db.Open(config.DBPath)
	if err != nil {
		log.Fatal("Error opening database").Err(err).Send()
	}
}
