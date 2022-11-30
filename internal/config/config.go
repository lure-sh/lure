package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	RootCmd string `toml:"rootCmd"`
	Repos   []Repo `toml:"repo"`
}

type Repo struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}

var defaultConfig = Config{
	RootCmd: "sudo",
	Repos: []Repo{
		{
			Name: "default",
			URL:  "https://github.com/Arsen6331/lure-repo.git",
		},
	},
}

func Decode(cfg *Config) error {
	cfgFl, err := os.Open(ConfigPath)
	if err != nil {
		return err
	}
	defer cfgFl.Close()

	return toml.NewDecoder(cfgFl).Decode(cfg)
}
