package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/distro"
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
		vars, err := getBuildVars(c.Context, script, info)
		if err != nil {
			log.Fatal("Error getting build variables").Err(err).Send()
		}

		fmt.Println(vars.Name, vars.Version)
	}

	return nil
}
