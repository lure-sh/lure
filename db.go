package main

import (
	"github.com/genjidb/genji"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
)

var gdb *genji.DB

func init() {
	var err error
	gdb, err = genji.Open(config.DBPath)
	if err != nil {
		log.Fatal("Error opening database").Err(err).Send()
	}

	err = db.Init(gdb)
	if err != nil {
		log.Fatal("Error initializing database").Err(err).Send()
	}
}
