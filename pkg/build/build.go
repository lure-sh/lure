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

package build

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "github.com/goreleaser/nfpm/v2/apk"
	_ "github.com/goreleaser/nfpm/v2/arch"
	_ "github.com/goreleaser/nfpm/v2/deb"
	_ "github.com/goreleaser/nfpm/v2/rpm"

	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"lure.sh/lure/internal/cliutils"
	"lure.sh/lure/internal/config"
	"lure.sh/lure/internal/cpu"
	"lure.sh/lure/internal/db"
	"lure.sh/lure/internal/dl"
	"lure.sh/lure/internal/shutils/decoder"
	"lure.sh/lure/internal/shutils/handlers"
	"lure.sh/lure/internal/shutils/helpers"
	"lure.sh/lure/internal/types"
	"lure.sh/lure/pkg/distro"
	"lure.sh/lure/pkg/loggerctx"
	"lure.sh/lure/pkg/manager"
	"lure.sh/lure/pkg/repos"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// BuildPackage builds the script at the given path. It returns two slices. One contains the paths
// to the built package(s), the other contains the names of the built package(s).
func BuildPackage(ctx context.Context, opts types.BuildOpts) ([]string, []string, error) {
	log := loggerctx.From(ctx)

	info, err := distro.ParseOSRelease(ctx)
	if err != nil {
		return nil, nil, err
	}

	fl, err := parseScript(info, opts.Script)
	if err != nil {
		return nil, nil, err
	}

	// The first pass is just used to get variable values and runs before
	// the script is displayed, so it's restricted so as to prevent malicious
	// code from executing.
	vars, err := executeFirstPass(ctx, info, fl, opts.Script)
	if err != nil {
		return nil, nil, err
	}

	dirs := getDirs(ctx, vars, opts.Script)

	// If opts.Clean isn't set and we find the package already built,
	// just return it rather than rebuilding
	if !opts.Clean {
		builtPkgPath, ok, err := checkForBuiltPackage(opts.Manager, vars, getPkgFormat(opts.Manager), dirs.BaseDir)
		if err != nil {
			return nil, nil, err
		}

		if ok {
			return []string{builtPkgPath}, nil, err
		}
	}

	// Ask the user if they'd like to see the build script
	err = cliutils.PromptViewScript(ctx, opts.Script, vars.Name, config.Config(ctx).PagerStyle, opts.Interactive)
	if err != nil {
		log.Fatal("Failed to prompt user to view build script").Err(err).Send()
	}

	log.Info("Building package").Str("name", vars.Name).Str("version", vars.Version).Send()

	// The second pass will be used to execute the actual code,
	// so it's unrestricted. The script has already been displayed
	// to the user by this point, so it should be safe
	dec, err := executeSecondPass(ctx, info, fl, dirs)
	if err != nil {
		return nil, nil, err
	}

	// Get the installed packages on the system
	installed, err := opts.Manager.ListInstalled(nil)
	if err != nil {
		return nil, nil, err
	}

	cont, err := performChecks(ctx, vars, opts.Interactive, installed)
	if err != nil {
		return nil, nil, err
	} else if !cont {
		os.Exit(1)
	}

	// Prepare the directories for building
	err = prepareDirs(dirs)
	if err != nil {
		return nil, nil, err
	}

	buildDeps, err := installBuildDeps(ctx, vars, opts, installed)
	if err != nil {
		return nil, nil, err
	}

	err = installOptDeps(ctx, vars, opts, installed)
	if err != nil {
		return nil, nil, err
	}

	builtPaths, builtNames, repoDeps, err := buildLUREDeps(ctx, opts, vars)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Downloading sources").Send()

	err = getSources(ctx, dirs, vars)
	if err != nil {
		return nil, nil, err
	}

	err = executeFunctions(ctx, dec, dirs, vars)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Building package metadata").Str("name", vars.Name).Send()

	pkgFormat := getPkgFormat(opts.Manager)

	pkgInfo, err := buildPkgMetadata(vars, dirs, pkgFormat, append(repoDeps, builtNames...))
	if err != nil {
		return nil, nil, err
	}

	packager, err := nfpm.Get(pkgFormat)
	if err != nil {
		return nil, nil, err
	}

	pkgName := packager.ConventionalFileName(pkgInfo)
	pkgPath := filepath.Join(dirs.BaseDir, pkgName)

	pkgFile, err := os.Create(pkgPath)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Compressing package").Str("name", pkgName).Send()

	err = packager.Package(pkgInfo, pkgFile)
	if err != nil {
		return nil, nil, err
	}

	err = removeBuildDeps(ctx, buildDeps, opts)
	if err != nil {
		return nil, nil, err
	}

	// Add the path and name of the package we just built to the
	// appropriate slices
	pkgPaths := append(builtPaths, pkgPath)
	pkgNames := append(builtNames, vars.Name)

	// Remove any duplicates from the pkgPaths and pkgNames.
	// Duplicates can be introduced if several of the dependencies
	// depend on the same packages.
	pkgPaths = removeDuplicates(pkgPaths)
	pkgNames = removeDuplicates(pkgNames)

	return pkgPaths, pkgNames, nil
}

// parseScript parses the build script using the built-in bash implementation
func parseScript(info *distro.OSRelease, script string) (*syntax.File, error) {
	fl, err := os.Open(script)
	if err != nil {
		return nil, err
	}
	defer fl.Close()

	file, err := syntax.NewParser().Parse(fl, "lure.sh")
	if err != nil {
		return nil, err
	}

	return file, nil
}

// executeFirstPass executes the parsed script in a restricted environment
// to extract the build variables without executing any actual code.
func executeFirstPass(ctx context.Context, info *distro.OSRelease, fl *syntax.File, script string) (*types.BuildVars, error) {
	scriptDir := filepath.Dir(script)
	env := createBuildEnvVars(info, types.Directories{ScriptDir: scriptDir})

	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(helpers.Restricted.ExecHandler(handlers.NopExec)),
		interp.ReadDirHandler(handlers.RestrictedReadDir(scriptDir)),
		interp.StatHandler(handlers.RestrictedStat(scriptDir)),
		interp.OpenHandler(handlers.RestrictedOpen(scriptDir)),
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, fl)
	if err != nil {
		return nil, err
	}

	dec := decoder.New(info, runner)

	var vars types.BuildVars
	err = dec.DecodeVars(&vars)
	if err != nil {
		return nil, err
	}

	return &vars, nil
}

// getDirs returns the appropriate directories for the script
func getDirs(ctx context.Context, vars *types.BuildVars, script string) types.Directories {
	baseDir := filepath.Join(config.GetPaths(ctx).PkgsDir, vars.Name)
	return types.Directories{
		BaseDir:   baseDir,
		SrcDir:    filepath.Join(baseDir, "src"),
		PkgDir:    filepath.Join(baseDir, "pkg"),
		ScriptDir: filepath.Dir(script),
	}
}

// executeSecondPass executes the build script for the second time, this time without any restrictions.
// It returns a decoder that can be used to retrieve functions and variables from the script.
func executeSecondPass(ctx context.Context, info *distro.OSRelease, fl *syntax.File, dirs types.Directories) (*decoder.Decoder, error) {
	env := createBuildEnvVars(info, dirs)

	fakeroot := handlers.FakerootExecHandler(2 * time.Second)
	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(helpers.Helpers.ExecHandler(fakeroot)),
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, fl)
	if err != nil {
		return nil, err
	}

	return decoder.New(info, runner), nil
}

// prepareDirs prepares the directories for building.
func prepareDirs(dirs types.Directories) error {
	err := os.RemoveAll(dirs.BaseDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dirs.SrcDir, 0o755)
	if err != nil {
		return err
	}
	return os.MkdirAll(dirs.PkgDir, 0o755)
}

// performChecks checks various things on the system to ensure that the package can be installed.
func performChecks(ctx context.Context, vars *types.BuildVars, interactive bool, installed map[string]string) (bool, error) {
	log := loggerctx.From(ctx)
	if !cpu.IsCompatibleWith(cpu.Arch(), vars.Architectures) {
		cont, err := cliutils.YesNoPrompt(ctx, "Your system's CPU architecture doesn't match this package. Do you want to build anyway?", interactive, true)
		if err != nil {
			return false, err
		}

		if !cont {
			return false, nil
		}
	}

	if instVer, ok := installed[vars.Name]; ok {
		log.Warn("This package is already installed").
			Str("name", vars.Name).
			Str("version", instVer).
			Send()
	}

	return true, nil
}

// installBuildDeps installs any build dependencies that aren't already installed and returns
// a slice containing the names of all the packages it installed.
func installBuildDeps(ctx context.Context, vars *types.BuildVars, opts types.BuildOpts, installed map[string]string) ([]string, error) {
	log := loggerctx.From(ctx)
	var buildDeps []string
	if len(vars.BuildDepends) > 0 {
		found, notFound, err := repos.FindPkgs(ctx, vars.BuildDepends)
		if err != nil {
			return nil, err
		}

		found = removeAlreadyInstalled(found, installed)

		log.Info("Installing build dependencies").Send()

		flattened := cliutils.FlattenPkgs(ctx, found, "install", opts.Interactive)
		buildDeps = packageNames(flattened)
		InstallPkgs(ctx, flattened, notFound, opts)
	}
	return buildDeps, nil
}

// installOptDeps asks the user which, if any, optional dependencies they want to install.
// If the user chooses to install any optional dependencies, it performs the installation.
func installOptDeps(ctx context.Context, vars *types.BuildVars, opts types.BuildOpts, installed map[string]string) error {
	if len(vars.OptDepends) > 0 {
		optDeps, err := cliutils.ChooseOptDepends(ctx, vars.OptDepends, "install", opts.Interactive)
		if err != nil {
			return err
		}

		if len(optDeps) == 0 {
			return nil
		}

		found, notFound, err := repos.FindPkgs(ctx, optDeps)
		if err != nil {
			return err
		}

		found = removeAlreadyInstalled(found, installed)
		flattened := cliutils.FlattenPkgs(ctx, found, "install", opts.Interactive)
		InstallPkgs(ctx, flattened, notFound, opts)
	}
	return nil
}

// buildLUREDeps builds all the LURE dependencies of the package. It returns the paths and names
// of the packages it built, as well as all the dependencies it didn't find in the LURE repo so
// they can be installed from the system repos.
func buildLUREDeps(ctx context.Context, opts types.BuildOpts, vars *types.BuildVars) (builtPaths, builtNames, repoDeps []string, err error) {
	log := loggerctx.From(ctx)
	if len(vars.Depends) > 0 {
		log.Info("Installing dependencies").Send()

		found, notFound, err := repos.FindPkgs(ctx, vars.Depends)
		if err != nil {
			return nil, nil, nil, err
		}
		repoDeps = notFound

		// If there are multiple options for some packages, flatten them all into a single slice
		pkgs := cliutils.FlattenPkgs(ctx, found, "install", opts.Interactive)
		scripts := GetScriptPaths(ctx, pkgs)
		for _, script := range scripts {
			newOpts := opts
			newOpts.Script = script

			// Build the dependency
			pkgPaths, pkgNames, err := BuildPackage(ctx, newOpts)
			if err != nil {
				return nil, nil, nil, err
			}

			// Append the paths of all the built packages to builtPaths
			builtPaths = append(builtPaths, pkgPaths...)
			// Append the names of all the built packages to builtNames
			builtNames = append(builtNames, pkgNames...)
			// Append the name of the current package to builtNames
			builtNames = append(builtNames, filepath.Base(filepath.Dir(script)))
		}
	}

	// Remove any potential duplicates, which can be introduced if
	// several of the dependencies depend on the same packages.
	repoDeps = removeDuplicates(repoDeps)
	builtPaths = removeDuplicates(builtPaths)
	builtNames = removeDuplicates(builtNames)
	return builtPaths, builtNames, repoDeps, nil
}

// executeFunctions executes the special LURE functions, such as version(), prepare(), etc.
func executeFunctions(ctx context.Context, dec *decoder.Decoder, dirs types.Directories, vars *types.BuildVars) (err error) {
	log := loggerctx.From(ctx)
	version, ok := dec.GetFunc("version")
	if ok {
		log.Info("Executing version()").Send()

		buf := &bytes.Buffer{}

		err = version(
			ctx,
			interp.Dir(dirs.SrcDir),
			interp.StdIO(os.Stdin, buf, os.Stderr),
		)
		if err != nil {
			return err
		}

		newVer := strings.TrimSpace(buf.String())
		err = setVersion(ctx, dec.Runner, newVer)
		if err != nil {
			return err
		}
		vars.Version = newVer

		log.Info("Updating version").Str("new", newVer).Send()
	}

	prepare, ok := dec.GetFunc("prepare")
	if ok {
		log.Info("Executing prepare()").Send()

		err = prepare(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return err
		}
	}

	build, ok := dec.GetFunc("build")
	if ok {
		log.Info("Executing build()").Send()

		err = build(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return err
		}
	}

	packageFn, ok := dec.GetFunc("package")
	if ok {
		log.Info("Executing package()").Send()

		err = packageFn(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return err
		}
	} else {
		log.Fatal("The package() function is required").Send()
	}

	return nil
}

// buildPkgMetadata builds the metadata for the package that's going to be built.
func buildPkgMetadata(vars *types.BuildVars, dirs types.Directories, pkgFormat string, deps []string) (*nfpm.Info, error) {
	pkgInfo := &nfpm.Info{
		Name:        vars.Name,
		Description: vars.Description,
		Arch:        cpu.Arch(),
		Platform:    "linux",
		Version:     vars.Version,
		Release:     strconv.Itoa(vars.Release),
		Homepage:    vars.Homepage,
		License:     strings.Join(vars.Licenses, ", "),
		Maintainer:  vars.Maintainer,
		Overridables: nfpm.Overridables{
			Conflicts: vars.Conflicts,
			Replaces:  vars.Replaces,
			Provides:  vars.Provides,
			Depends:   deps,
		},
	}

	if pkgFormat == "apk" {
		// Alpine refuses to install packages that provide themselves, so remove any such provides
		pkgInfo.Overridables.Provides = slices.DeleteFunc(pkgInfo.Overridables.Provides, func(s string) bool {
			return s == pkgInfo.Name
		})
	}

	if vars.Epoch != 0 {
		pkgInfo.Epoch = strconv.FormatUint(uint64(vars.Epoch), 10)
	}

	setScripts(vars, pkgInfo, dirs.ScriptDir)

	if slices.Contains(vars.Architectures, "all") {
		pkgInfo.Arch = "all"
	}

	contents, err := buildContents(vars, dirs)
	if err != nil {
		return nil, err
	}
	pkgInfo.Overridables.Contents = contents

	return pkgInfo, nil
}

// buildContents builds the contents section of the package, which contains the files
// that will be placed into the final package.
func buildContents(vars *types.BuildVars, dirs types.Directories) ([]*files.Content, error) {
	contents := []*files.Content{}
	err := filepath.Walk(dirs.PkgDir, func(path string, fi os.FileInfo, err error) error {
		trimmed := strings.TrimPrefix(path, dirs.PkgDir)

		if fi.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}

			// If the directory is empty, skip it
			_, err = f.Readdirnames(1)
			if err != io.EOF {
				return nil
			}

			contents = append(contents, &files.Content{
				Source:      path,
				Destination: trimmed,
				Type:        "dir",
				FileInfo: &files.ContentFileInfo{
					MTime: fi.ModTime(),
				},
			})

			return f.Close()
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			// Remove pkgdir from the symlink's path
			link = strings.TrimPrefix(link, dirs.PkgDir)

			contents = append(contents, &files.Content{
				Source:      link,
				Destination: trimmed,
				Type:        "symlink",
				FileInfo: &files.ContentFileInfo{
					MTime: fi.ModTime(),
					Mode:  fi.Mode(),
				},
			})

			return nil
		}

		fileContent := &files.Content{
			Source:      path,
			Destination: trimmed,
			FileInfo: &files.ContentFileInfo{
				MTime: fi.ModTime(),
				Mode:  fi.Mode(),
				Size:  fi.Size(),
			},
		}

		// If the file is supposed to be backed up, set its type to config|noreplace
		if slices.Contains(vars.Backup, trimmed) {
			fileContent.Type = "config|noreplace"
		}

		contents = append(contents, fileContent)

		return nil
	})
	return contents, err
}

// removeBuildDeps asks the user if they'd like to remove the build dependencies that were
// installed by installBuildDeps. If so, it uses the package manager to do that.
func removeBuildDeps(ctx context.Context, buildDeps []string, opts types.BuildOpts) error {
	if len(buildDeps) > 0 {
		remove, err := cliutils.YesNoPrompt(ctx, "Would you like to remove the build dependencies?", opts.Interactive, false)
		if err != nil {
			return err
		}

		if remove {
			err = opts.Manager.Remove(
				&manager.Opts{
					AsRoot:    true,
					NoConfirm: true,
				},
				buildDeps...,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// checkForBuiltPackage tries to detect a previously-built package and returns its path
// and true if it finds one. If it doesn't find it, it returns "", false, nil.
func checkForBuiltPackage(mgr manager.Manager, vars *types.BuildVars, pkgFormat, baseDir string) (string, bool, error) {
	filename, err := pkgFileName(vars, pkgFormat)
	if err != nil {
		return "", false, err
	}

	pkgPath := filepath.Join(baseDir, filename)

	_, err = os.Stat(pkgPath)
	if err != nil {
		return "", false, nil
	}

	return pkgPath, true, nil
}

// pkgFileName returns the filename of the package if it were to be built.
// This is used to check if the package has already been built.
func pkgFileName(vars *types.BuildVars, pkgFormat string) (string, error) {
	pkgInfo := &nfpm.Info{
		Name:    vars.Name,
		Arch:    cpu.Arch(),
		Version: vars.Version,
		Release: strconv.Itoa(vars.Release),
		Epoch:   strconv.FormatUint(uint64(vars.Epoch), 10),
	}

	packager, err := nfpm.Get(pkgFormat)
	if err != nil {
		return "", err
	}

	return packager.ConventionalFileName(pkgInfo), nil
}

// getPkgFormat returns the package format of the package manager,
// or LURE_PKG_FORMAT if that's set.
func getPkgFormat(mgr manager.Manager) string {
	pkgFormat := mgr.Format()
	if format, ok := os.LookupEnv("LURE_PKG_FORMAT"); ok {
		pkgFormat = format
	}
	return pkgFormat
}

// createBuildEnvVars creates the environment variables that will be set in the
// build script when it's executed.
func createBuildEnvVars(info *distro.OSRelease, dirs types.Directories) []string {
	env := os.Environ()

	env = append(
		env,
		"DISTRO_NAME="+info.Name,
		"DISTRO_PRETTY_NAME="+info.PrettyName,
		"DISTRO_ID="+info.ID,
		"DISTRO_VERSION_ID="+info.VersionID,
		"DISTRO_ID_LIKE="+strings.Join(info.Like, " "),
		"ARCH="+cpu.Arch(),
		"NCPU="+strconv.Itoa(runtime.NumCPU()),
	)

	if dirs.ScriptDir != "" {
		env = append(env, "scriptdir="+dirs.ScriptDir)
	}

	if dirs.PkgDir != "" {
		env = append(env, "pkgdir="+dirs.PkgDir)
	}

	if dirs.SrcDir != "" {
		env = append(env, "srcdir="+dirs.SrcDir)
	}

	return env
}

// getSources downloads the sources from the script.
func getSources(ctx context.Context, dirs types.Directories, bv *types.BuildVars) error {
	log := loggerctx.From(ctx)
	if len(bv.Sources) != len(bv.Checksums) {
		log.Fatal("The checksums array must be the same length as sources").Send()
	}

	for i, src := range bv.Sources {
		opts := dl.Options{
			Name:        fmt.Sprintf("%s[%d]", bv.Name, i),
			URL:         src,
			Destination: dirs.SrcDir,
			Progress:    os.Stderr,
			LocalDir:    dirs.ScriptDir,
		}

		if !strings.EqualFold(bv.Checksums[i], "SKIP") {
			// If the checksum contains a colon, use the part before the colon
			// as the algorithm and the part after as the actual checksum.
			// Otherwise, use the default sha256 with the whole string as the checksum.
			algo, hashData, ok := strings.Cut(bv.Checksums[i], ":")
			if ok {
				checksum, err := hex.DecodeString(hashData)
				if err != nil {
					return err
				}
				opts.Hash = checksum
				opts.HashAlgorithm = algo
			} else {
				checksum, err := hex.DecodeString(bv.Checksums[i])
				if err != nil {
					return err
				}
				opts.Hash = checksum
			}
		}

		err := dl.Download(ctx, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

// setScripts adds any hook scripts to the package metadata.
func setScripts(vars *types.BuildVars, info *nfpm.Info, scriptDir string) {
	if vars.Scripts.PreInstall != "" {
		info.Scripts.PreInstall = filepath.Join(scriptDir, vars.Scripts.PreInstall)
	}

	if vars.Scripts.PostInstall != "" {
		info.Scripts.PostInstall = filepath.Join(scriptDir, vars.Scripts.PostInstall)
	}

	if vars.Scripts.PreRemove != "" {
		info.Scripts.PreRemove = filepath.Join(scriptDir, vars.Scripts.PreRemove)
	}

	if vars.Scripts.PostRemove != "" {
		info.Scripts.PostRemove = filepath.Join(scriptDir, vars.Scripts.PostRemove)
	}

	if vars.Scripts.PreUpgrade != "" {
		info.ArchLinux.Scripts.PreUpgrade = filepath.Join(scriptDir, vars.Scripts.PreUpgrade)
		info.APK.Scripts.PreUpgrade = filepath.Join(scriptDir, vars.Scripts.PreUpgrade)
	}

	if vars.Scripts.PostUpgrade != "" {
		info.ArchLinux.Scripts.PostUpgrade = filepath.Join(scriptDir, vars.Scripts.PostUpgrade)
		info.APK.Scripts.PostUpgrade = filepath.Join(scriptDir, vars.Scripts.PostUpgrade)
	}

	if vars.Scripts.PreTrans != "" {
		info.RPM.Scripts.PreTrans = filepath.Join(scriptDir, vars.Scripts.PreTrans)
	}

	if vars.Scripts.PostTrans != "" {
		info.RPM.Scripts.PostTrans = filepath.Join(scriptDir, vars.Scripts.PostTrans)
	}
}

// setVersion changes the version variable in the script runner.
// It's used to set the version to the output of the version() function.
func setVersion(ctx context.Context, r *interp.Runner, to string) error {
	fl, err := syntax.NewParser().Parse(strings.NewReader("version='"+to+"'"), "")
	if err != nil {
		return err
	}
	return r.Run(ctx, fl)
}

// removeAlreadyInstalled returns a map without any dependencies that are already installed
func removeAlreadyInstalled(found map[string][]db.Package, installed map[string]string) map[string][]db.Package {
	filteredPackages := make(map[string][]db.Package)

	for name, pkgList := range found {
		filteredPkgList := []db.Package{}
		for _, pkg := range pkgList {
			if _, isInstalled := installed[pkg.Name]; !isInstalled {
				filteredPkgList = append(filteredPkgList, pkg)
			}
		}
		filteredPackages[name] = filteredPkgList
	}

	return filteredPackages
}

// packageNames returns the names of all the given packages
func packageNames(pkgs []db.Package) []string {
	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = p.Name
	}
	return names
}

// removeDuplicates removes any duplicates from the given slice
func removeDuplicates(slice []string) []string {
	seen := map[string]struct{}{}
	result := []string{}

	for _, s := range slice {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}
