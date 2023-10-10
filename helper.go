package main

import (
	"os"

	"github.com/urfave/cli/v2"
	"lure.sh/lure/internal/shutils/helpers"
	"lure.sh/lure/pkg/loggerctx"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

var helperCmd = &cli.Command{
	Name:  "helper",
	Usage: "Run a lure helper command",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dest-dir",
			Aliases: []string{"d"},
			Usage:   "The directory that the install commands will install to",
			Value:   "dest",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		log := loggerctx.From(ctx)

		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Error getting working directory").Err(err).Send()
		}

		helper, ok := helpers.Helpers[c.Args().First()]
		if !ok {
			log.Fatal("No such helper command").Str("handler", c.Args().First()).Send()
		}

		hc := interp.HandlerContext{
			Env:    expand.ListEnviron("pkgdir=" + c.String("dest-dir")),
			Dir:    wd,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		return helper(hc, c.Args().First(), c.Args().Slice()[1:])
	},
}
