package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
	"go.arsenm.dev/lure/internal/types"
)

var defaultConfig = types.Config{
	RootCmd: "sudo",
	Repos: []types.Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	},
}

func Decode(cfg *types.Config) error {
	cfgFl, err := os.Open(ConfigPath)
	if err != nil {
		return err
	}
	defer cfgFl.Close()

	return toml.NewDecoder(cfgFl).Decode(cfg)
}
