package types

type RepoConfig struct {
	Repo struct {
		MinVersion string `toml:"minVersion"`
	}
}
