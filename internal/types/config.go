package types

// Config represents the LURE configuration file
type Config struct {
	RootCmd    string `toml:"rootCmd"`
	PagerStyle string `toml:"pagerStyle"`
	Repos      []Repo `toml:"repo"`
}

// Repo represents a LURE repo within a configuration file
type Repo struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}
