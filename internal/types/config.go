/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Elara Musayelyan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package types

// Config represents the LURE configuration file
type Config struct {
	RootCmd          string   `toml:"rootCmd"`
	PagerStyle       string   `toml:"pagerStyle"`
	IgnorePkgUpdates []string `toml:"ignorePkgUpdates"`
	Repos            []Repo   `toml:"repo"`
	Unsafe           Unsafe   `toml:"unsafe"`
}

// Repo represents a LURE repo within a configuration file
type Repo struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}

type Unsafe struct {
	AllowRunAsRoot bool `toml:"allowRunAsRoot"`
}
