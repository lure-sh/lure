package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
	"go.arsenm.dev/lure/internal/types"
)

var defaultConfig = types.Config{
	RootCmd:    "sudo",
	PagerStyle: "native",
	Repos: []types.Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	},
}

// Decode decodes the config file into the given
// pointer
func Decode(cfg *types.Config) error {
	cfgFl, err := os.Open(ConfigPath)
	if err != nil {
		return err
	}
	defer cfgFl.Close()

	// Write defaults to pointer in case some values are not set in the config
	*cfg = defaultConfig
	// Set repos to nil so as to avoid a duplicate default
	cfg.Repos = nil
	return toml.NewDecoder(cfgFl).Decode(cfg)
}
