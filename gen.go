package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"lure.sh/lure/pkg/gen"
)

var genCmd = &cli.Command{
	Name:    "generate",
	Usage:   "Generate a LURE script from a template",
	Aliases: []string{"gen"},
	Subcommands: []*cli.Command{
		genPipCmd,
	},
}

var genPipCmd = &cli.Command{
	Name:  "pip",
	Usage: "Generate a LURE script for a pip module",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Required: true,
		},
		&cli.StringFlag{
			Name:     "version",
			Aliases:  []string{"v"},
			Required: true,
		},
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Value:   "A Python Pip module",
		},
	},
	Action: func(c *cli.Context) error {
		return gen.Pip(os.Stdout, gen.PipOptions{
			Name:        c.String("name"),
			Version:     c.String("version"),
			Description: c.String("description"),
		})
	},
}
