package build

import (
	"context"
	"path/filepath"

	"go.elara.ws/logger/log"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/internal/types"
)

// InstallPkgs installs non-LURE packages via the package manager, then builds and installs LURE
// packages
func InstallPkgs(ctx context.Context, lurePkgs []db.Package, nativePkgs []string, opts types.BuildOpts) {
	if len(nativePkgs) > 0 {
		err := opts.Manager.Install(nil, nativePkgs...)
		if err != nil {
			log.Fatal("Error installing native packages").Err(err).Send()
		}
	}

	InstallScripts(ctx, GetScriptPaths(lurePkgs), opts)
}

// GetScriptPaths generates a slice of script paths corresponding to the
// given packages
func GetScriptPaths(pkgs []db.Package) []string {
	var scripts []string
	for _, pkg := range pkgs {
		scriptPath := filepath.Join(config.GetPaths().RepoDir, pkg.Repository, pkg.Name, "lure.sh")
		scripts = append(scripts, scriptPath)
	}
	return scripts
}

// InstallScripts builds and installs LURE build scripts
func InstallScripts(ctx context.Context, scripts []string, opts types.BuildOpts) {
	for _, script := range scripts {
		opts.Script = script
		builtPkgs, _, err := BuildPackage(ctx, opts)
		if err != nil {
			log.Fatal("Error building package").Err(err).Send()
		}

		err = opts.Manager.InstallLocal(nil, builtPkgs...)
		if err != nil {
			log.Fatal("Error installing package").Err(err).Send()
		}
	}
}
