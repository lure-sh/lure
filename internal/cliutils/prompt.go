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

package cliutils

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/internal/log"
	"go.elara.ws/lure/internal/pager"
	"go.elara.ws/lure/internal/translations"
)

// YesNoPrompt asks the user a yes or no question, using def as the default answer
func YesNoPrompt(msg string, interactive, def bool) (bool, error) {
	if interactive {
		var answer bool
		err := survey.AskOne(
			&survey.Confirm{
				Message: translations.Translator().TranslateTo(msg, config.Language()),
				Default: def,
			},
			&answer,
		)
		return answer, err
	} else {
		return def, nil
	}
}

// PromptViewScript asks the user if they'd like to see a script,
// shows it if they answer yes, then asks if they'd still like to
// continue, and exits if they answer no.
func PromptViewScript(script, name, style string, interactive bool) error {
	if !interactive {
		return nil
	}

	scriptPrompt := translations.Translator().TranslateTo("Would you like to view the build script for", config.Language()) + " " + name
	view, err := YesNoPrompt(scriptPrompt, interactive, false)
	if err != nil {
		return err
	}

	if view {
		err = ShowScript(script, name, style)
		if err != nil {
			return err
		}

		cont, err := YesNoPrompt("Would you still like to continue?", interactive, false)
		if err != nil {
			return err
		}

		if !cont {
			log.Fatal(translations.Translator().TranslateTo("User chose not to continue after reading script", config.Language())).Send()
		}
	}

	return nil
}

// ShowScript uses the built-in pager to display a script at a
// given path, in the given syntax highlighting style.
func ShowScript(path, name, style string) error {
	scriptFl, err := os.Open(path)
	if err != nil {
		return err
	}
	defer scriptFl.Close()

	str, err := pager.SyntaxHighlightBash(scriptFl, style)
	if err != nil {
		return err
	}

	pgr := pager.New(name, str)
	return pgr.Run()
}

// FlattenPkgs attempts to flatten the a map of slices of packages into a single slice
// of packages by prompting the user if multiple packages match.
func FlattenPkgs(found map[string][]db.Package, verb string, interactive bool) []db.Package {
	var outPkgs []db.Package
	for _, pkgs := range found {
		if len(pkgs) > 1 && interactive {
			choices, err := PkgPrompt(pkgs, verb, interactive)
			if err != nil {
				log.Fatal("Error prompting for choice of package").Send()
			}
			outPkgs = append(outPkgs, choices...)
		} else if len(pkgs) == 1 || !interactive {
			outPkgs = append(outPkgs, pkgs[0])
		}
	}
	return outPkgs
}

// PkgPrompt asks the user to choose between multiple packages.
// The user may choose multiple packages.
func PkgPrompt(options []db.Package, verb string, interactive bool) ([]db.Package, error) {
	if !interactive {
		return []db.Package{options[0]}, nil
	}

	names := make([]string, len(options))
	for i, option := range options {
		names[i] = option.Repository + "/" + option.Name + " " + option.Version
	}

	prompt := &survey.MultiSelect{
		Options: names,
		Message: translations.Translator().TranslateTo("Choose which package(s) to "+verb, config.Language()),
	}

	var choices []int
	err := survey.AskOne(prompt, &choices)
	if err != nil {
		return nil, err
	}

	out := make([]db.Package, len(choices))
	for i, choiceIndex := range choices {
		out[i] = options[choiceIndex]
	}

	return out, nil
}
