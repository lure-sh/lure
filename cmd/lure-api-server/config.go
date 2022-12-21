package main

import (
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/types"
)

var cfg types.Config

func init() {
	err := config.Decode(&cfg)
	if err != nil {
		log.Fatal("Error decoding config file").Err(err).Send()
	}
}
