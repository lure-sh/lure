package types

type Config struct {
	RootCmd string `toml:"rootCmd"`
	Repos   []Repo `toml:"repo"`
}

type Repo struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}
