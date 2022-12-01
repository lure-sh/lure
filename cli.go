package main

import (
	"github.com/AlecAivazis/survey/v2"
	"go.arsenm.dev/lure/internal/db"
)

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
