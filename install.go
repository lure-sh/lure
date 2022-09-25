package main

import (
	"context"

	"github.com/urfave/cli/v2"
	"go.arsenm.dev/lure/manager"
)

func installCmd(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 1 {
		log.Fatalf("Command install expected at least 1 argument, got %d", args.Len()).Send()
	}

	mgr := manager.Detect()
	if mgr == nil {
		log.Fatal("Unable to detect supported package manager on system").Send()
	}

	installPkgs(c.Context, args.Slice(), mgr)

	return nil
}

func installPkgs(ctx context.Context, pkgs []string, mgr manager.Manager) {
	err := pullRepos(ctx)
	if err != nil {
		log.Fatal("Error pulling repositories").Err(err).Send()
	}

	scripts, notFound := findPkgs(pkgs)

	if len(notFound) > 0 {
		err = mgr.Install(notFound...)
		if err != nil {
			log.Fatal("Error installing native packages").Err(err).Send()
		}
	}

	installScripts(ctx, mgr, scripts)
}

func installScripts(ctx context.Context, mgr manager.Manager, scripts []string) {
	for _, script := range scripts {
		builtPkgs, _, err := buildPackage(ctx, script, mgr)
		if err != nil {
			log.Fatal("Error building package").Err(err).Send()
		}

		err = mgr.InstallLocal(builtPkgs...)
		if err != nil {
			log.Fatal("Error installing package").Err(err).Send()
		}
	}
}

func removeCmd(c *cli.Context) error {
	args := c.Args()
	if args.Len() < 1 {
		log.Fatalf("Command remove expected at least 1 argument, got %d", args.Len()).Send()
	}

	mgr := manager.Detect()
	if mgr == nil {
		log.Fatal("Unable to detect supported package manager on system").Send()
	}

	err := mgr.Remove(c.Args().Slice()...)
	if err != nil {
		log.Fatal("Error removing packages").Err(err).Send()
	}

	return nil
}
