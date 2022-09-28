package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/internal/shutils"
	"go.arsenm.dev/lure/internal/shutils/decoder"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func listCmd(c *cli.Context) error {
	info, err := distro.ParseOSRelease(c.Context)
	if err != nil {
		log.Fatal("Error parsing os-release").Err(err).Send()
	}

	pkgs, err := findPkg("*")
	if err != nil {
		log.Fatal("Error finding packages").Err(err).Send()
	}

	for _, script := range pkgs {
		fl, err := os.Open(script)
		if err != nil {
			log.Fatal("Error opening script").Err(err).Send()
		}

		file, err := syntax.NewParser().Parse(fl, "lure.sh")
		if err != nil {
			log.Fatal("Error parsing script").Err(err).Send()
		}

		fl.Close()

		runner, err := interp.New(
			interp.Env(expand.ListEnviron()),
			interp.ExecHandler(shutils.NopExec),
			interp.StatHandler(shutils.NopStat),
			interp.OpenHandler(shutils.NopOpen),
			interp.ReadDirHandler(shutils.NopReadDir),
		)
		if err != nil {
			log.Fatal("Error creating runner").Err(err).Send()
		}

		err = runner.Run(c.Context, file)
		if err != nil {
			log.Fatal("Error running script").Err(err).Send()
		}

		dec := decoder.New(info, runner)

		var vars BuildVars
		err = dec.DecodeVars(&vars)
		if err != nil {
			log.Fatal("Error decoding script variables").Err(err).Send()
		}

		fmt.Println(vars.Name, vars.Version)
	}

	return nil
}
