package types

// RepoConfig represents a LURE repo's lure-repo.toml file.
type RepoConfig struct {
	Repo struct {
		MinVersion string `toml:"minVersion"`
	}
}
