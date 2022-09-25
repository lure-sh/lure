package main

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"go.arsenm.dev/lure/manager"
)

var (
	cacheDir string
	cfgPath  string
	config   Config
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

func init() {
	cfg, cache, err := makeDirs()
	if err != nil {
		log.Fatal("Error creating directories").Err(err).Send()
	}
	cacheDir = cache

	cfgPath = filepath.Join(cfg, "lure.toml")

	cfgFl, err := os.Open(cfgPath)
	if err != nil {
		log.Fatal("Error opening config file").Err(err).Send()
	}
	defer cfgFl.Close()

	err = toml.NewDecoder(cfgFl).Decode(&config)
	if err != nil {
		log.Fatal("Error decoding config file").Err(err).Send()
	}

	manager.DefaultRootCmd = config.RootCmd
}

func makeDirs() (string, string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", "", err
	}

	baseCfgPath := filepath.Join(cfgDir, "lure")

	err = os.MkdirAll(baseCfgPath, 0o755)
	if err != nil {
		return "", "", err
	}

	cfgPath := filepath.Join(baseCfgPath, "lure.toml")

	if _, err := os.Stat(cfgPath); err != nil {
		cfgFl, err := os.Create(cfgPath)
		if err != nil {
			return "", "", err
		}

		err = toml.NewEncoder(cfgFl).Encode(&defaultConfig)
		if err != nil {
			return "", "", err
		}

		cfgFl.Close()
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", "", err
	}

	baseCachePath := filepath.Join(cacheDir, "lure")

	err = os.MkdirAll(filepath.Join(baseCachePath, "repo"), 0o755)
	if err != nil {
		return "", "", err
	}

	err = os.MkdirAll(filepath.Join(baseCachePath, "pkgs"), 0o755)
	if err != nil {
		return "", "", err
	}

	return baseCfgPath, baseCachePath, nil
}
