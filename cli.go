package main

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/internal/db"
	"go.arsenm.dev/lure/internal/pager"
)

// pkgPrompt asks the user to choose between multiple packages.
// The user may choose multiple packages.
func pkgPrompt(options []db.Package, verb string) ([]db.Package, error) {
	names := make([]string, len(options))
	for i, option := range options {
		names[i] = option.Repository + "/" + option.Name + " " + option.Version
	}

	prompt := &survey.MultiSelect{
		Options: names,
		Message: "Choose which package(s) to " + verb,
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

// yesNoPrompt asks the user a yes or no question, using def as the default answer
func yesNoPrompt(msg string, def bool) (bool, error) {
	var answer bool
	err := survey.AskOne(
		&survey.Confirm{
			Message: msg,
			Default: def,
		},
		&answer,
	)
	return answer, err
}

func promptViewScript(script string, name string) error {
	view, err := yesNoPrompt("Would you like to view the build script for "+name, false)
	if err != nil {
		return err
	}

	if view {
		err = showScript(script, name)
		if err != nil {
			return err
		}

		cont, err := yesNoPrompt("Would you still like to continue?", false)
		if err != nil {
			return err
		}

		if !cont {
			log.Fatal("User chose not to continue after reading script").Send()
		}
	}

	return nil
}

func showScript(path, name string) error {
	scriptFl, err := os.Open(path)
	if err != nil {
		return err
	}
	defer scriptFl.Close()

	str, err := pager.SyntaxHighlightBash(scriptFl, cfg.PagerStyle)
	if err != nil {
		return err
	}

	pgr := pager.New(name, str)
	return pgr.Run()
}
