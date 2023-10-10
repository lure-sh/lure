package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"lure.sh/lure/internal/cpu"
	"lure.sh/lure/internal/shutils/helpers"
	"lure.sh/lure/pkg/distro"
	"lure.sh/lure/pkg/loggerctx"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

var helperCmd = &cli.Command{
	Name:        "helper",
	Usage:       "Run a LURE helper command",
	ArgsUsage:   `<helper_name|"list">`,
	Subcommands: []*cli.Command{helperListCmd},
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

		if c.Args().Len() < 1 {
			cli.ShowSubcommandHelpAndExit(c, 1)
		}

		helper, ok := helpers.Helpers[c.Args().First()]
		if !ok {
			log.Fatal("No such helper command").Str("name", c.Args().First()).Send()
		}

		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Error getting working directory").Err(err).Send()
		}

		info, err := distro.ParseOSRelease(ctx)
		if err != nil {
			log.Fatal("Error getting working directory").Err(err).Send()
		}

		hc := interp.HandlerContext{
			Env: expand.ListEnviron(
				"pkgdir="+c.String("dest-dir"),
				"DISTRO_ID="+info.ID,
				"DISTRO_ID_LIKE="+strings.Join(info.Like, " "),
				"ARCH="+cpu.Arch(),
			),
			Dir:    wd,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		return helper(hc, c.Args().First(), c.Args().Slice()[1:])
	},
	CustomHelpTemplate: cli.CommandHelpTemplate,
	BashComplete: func(ctx *cli.Context) {
		for name := range helpers.Helpers {
			fmt.Println(name)
		}
	},
}

var helperListCmd = &cli.Command{
	Name:    "list",
	Usage:   "List all the available helper commands",
	Aliases: []string{"ls"},
	Action: func(ctx *cli.Context) error {
		for name := range helpers.Helpers {
			fmt.Println(name)
		}
		return nil
	},
}
