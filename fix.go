package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/repos"
)

func fixCmd(c *cli.Context) error {
	gdb.Close()

	log.Info("Removing cache directory").Send()

	err := os.RemoveAll(config.CacheDir)
	if err != nil {
		log.Fatal("Unable to remove cache directory").Err(err).Send()
	}

	log.Info("Rebuilding cache").Send()

	err = os.MkdirAll(config.CacheDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create new cache directory").Err(err).Send()
	}

	// Make sure the DB is rebuilt when repos are pulled
	gdb, err = db.Open(config.DBPath)
	if err != nil {
		log.Fatal("Error initializing database").Err(err).Send()
	}
	config.DBPresent = false

	err = repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repos").Err(err).Send()
	}

	log.Info("Done").Send()

	return nil
}
