/*
 * LURE - Linux User REpository
 * Copyright (C) 2022 Arsen Musayelyan
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

package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	_ "github.com/goreleaser/nfpm/v2/apk"
	_ "github.com/goreleaser/nfpm/v2/arch"
	_ "github.com/goreleaser/nfpm/v2/deb"
	_ "github.com/goreleaser/nfpm/v2/rpm"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"go.arsenm.dev/logger/log"
	"go.arsenm.dev/lure/distro"
	"go.arsenm.dev/lure/download"
	"go.arsenm.dev/lure/internal/config"
	"go.arsenm.dev/lure/internal/cpu"
	"go.arsenm.dev/lure/internal/repos"
	"go.arsenm.dev/lure/internal/shutils"
	"go.arsenm.dev/lure/internal/shutils/decoder"
	"go.arsenm.dev/lure/manager"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// BuildVars represents the script variables required
// to build a package
type BuildVars struct {
	Name          string   `sh:"name,required"`
	Version       string   `sh:"version,required"`
	Release       int      `sh:"release,required"`
	Epoch         uint     `sh:"epoch"`
	Description   string   `sh:"desc"`
	Homepage      string   `sh:"homepage"`
	Maintainer    string   `sh:"maintainer"`
	Architectures []string `sh:"architectures"`
	Licenses      []string `sh:"license"`
	Provides      []string `sh:"provides"`
	Conflicts     []string `sh:"conflicts"`
	Depends       []string `sh:"deps"`
	BuildDepends  []string `sh:"build_deps"`
	Replaces      []string `sh:"replaces"`
	Sources       []string `sh:"sources"`
	Checksums     []string `sh:"checksums"`
	Backup        []string `sh:"backup"`
	Scripts       Scripts  `sh:"scripts"`
}

type Scripts struct {
	PreInstall  string `sh:"preinstall"`
	PostInstall string `sh:"postinstall"`
	PreRemove   string `sh:"preremove"`
	PostRemove  string `sh:"postremove"`
	PreUpgrade  string `sh:"preupgrade"`
	PostUpgrade string `sh:"postupgrade"`
	PreTrans    string `sh:"pretrans"`
	PostTrans   string `sh:"posttrans"`
}

func buildCmd(c *cli.Context) error {
	script := c.String("script")

	err := repos.Pull(c.Context, gdb, cfg.Repos)
	if err != nil {
		log.Fatal("Error pulling repositories").Err(err).Send()
	}

	mgr := manager.Detect()
	if mgr == nil {
		log.Fatal("Unable to detect supported package manager on system").Send()
	}

	pkgPaths, _, err := buildPackage(c.Context, script, mgr)
	if err != nil {
		log.Fatal("Error building package").Err(err).Send()
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("Error getting working directory").Err(err).Send()
	}

	for _, pkgPath := range pkgPaths {
		name := filepath.Base(pkgPath)
		err = os.Rename(pkgPath, filepath.Join(wd, name))
		if err != nil {
			log.Fatal("Error moving the package").Err(err).Send()
		}
	}

	return nil
}

// buildPackage builds the script at the given path. It returns two slices. One contains the paths
// to the built package(s), the other contains the names of the built package(s).
func buildPackage(ctx context.Context, script string, mgr manager.Manager) ([]string, []string, error) {
	info, err := distro.ParseOSRelease(ctx)
	if err != nil {
		return nil, nil, err
	}

	var distroChanged bool
	if distID, ok := os.LookupEnv("LURE_DISTRO"); ok {
		info.ID = distID
		// Since the distro was overwritten, we don't know what the
		// like distros are, so set to nil
		info.Like = nil
		distroChanged = true
	}

	fl, err := os.Open(script)
	if err != nil {
		return nil, nil, err
	}

	file, err := syntax.NewParser().Parse(fl, "lure.sh")
	if err != nil {
		return nil, nil, err
	}

	fl.Close()

	env := genBuildEnv(info)

	scriptDir := filepath.Dir(script)

	// The first pass is just used to get variable values and runs before
	// the script is displayed, so it is restricted so as to prevent malicious
	// code from executing.
	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(rHelpers.ExecHandler(shutils.RestrictedExec("source"))),
		interp.ReadDirHandler(shutils.RestrictedReadDir(scriptDir)),
		interp.StatHandler(shutils.RestrictedStat(scriptDir)),
		interp.OpenHandler(shutils.RestrictedOpen(scriptDir)),
	)
	if err != nil {
		return nil, nil, err
	}

	err = runner.Run(ctx, file)
	if err != nil {
		return nil, nil, err
	}

	dec := decoder.New(info, runner)

	// If distro was changed, the list of like distros
	// no longer applies, so disable its use
	if distroChanged {
		dec.LikeDistros = false
	}

	var vars BuildVars
	err = dec.DecodeVars(&vars)
	if err != nil {
		return nil, nil, err
	}

	err = promptViewScript(script, vars.Name)
	if err != nil {
		log.Fatal("Failed to prompt user to view build script").Err(err).Send()
	}

	if !archMatches(vars.Architectures) {
		buildAnyway, err := yesNoPrompt("Your system's CPU architecture doesn't match this package. Do you want to build anyway?", true)
		if err != nil {
			return nil, nil, err
		}

		if !buildAnyway {
			os.Exit(1)
		}
	}

	log.Info("Building package").Str("name", vars.Name).Str("version", vars.Version).Send()

	// The second pass will be used to execute the actual functions,
	// so it cannot be restricted. The script has already been displayed
	// to the user by this point, so it should be safe
	runner, err = interp.New(
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(helpers.ExecHandler(nil)),
	)
	if err != nil {
		return nil, nil, err
	}

	err = runner.Run(ctx, file)
	if err != nil {
		return nil, nil, err
	}

	dec = decoder.New(info, runner)

	// If distro was changed, the list of like distros
	// no longer applies, so disable its use
	if distroChanged {
		dec.LikeDistros = false
	}

	baseDir := filepath.Join(config.PkgsDir, vars.Name)
	srcdir := filepath.Join(baseDir, "src")
	pkgdir := filepath.Join(baseDir, "pkg")

	err = os.RemoveAll(baseDir)
	if err != nil {
		return nil, nil, err
	}

	err = os.MkdirAll(srcdir, 0o755)
	if err != nil {
		return nil, nil, err
	}

	err = os.MkdirAll(pkgdir, 0o755)
	if err != nil {
		return nil, nil, err
	}

	installed, err := mgr.ListInstalled(nil)
	if err != nil {
		return nil, nil, err
	}

	var buildDeps []string
	for _, pkgName := range vars.BuildDepends {
		if _, ok := installed[pkgName]; !ok {
			buildDeps = append(buildDeps, pkgName)
		}
	}

	if len(buildDeps) > 0 {
		found, notFound, err := repos.FindPkgs(gdb, buildDeps)
		if err != nil {
			return nil, nil, err
		}

		log.Info("Installing build dependencies").Send()
		installPkgs(ctx, flattenFoundPkgs(found, "install"), notFound, mgr)
	}

	var builtDeps, builtNames, repoDeps []string
	if len(vars.Depends) > 0 {
		log.Info("Installing dependencies").Send()

		found, notFound, err := repos.FindPkgs(gdb, vars.Depends)
		if err != nil {
			return nil, nil, err
		}

		scripts := getScriptPaths(flattenFoundPkgs(found, "install"))
		for _, script := range scripts {
			pkgPaths, pkgNames, err := buildPackage(ctx, script, mgr)
			if err != nil {
				return nil, nil, err
			}
			builtDeps = append(builtDeps, pkgPaths...)
			builtNames = append(builtNames, pkgNames...)
			builtNames = append(builtNames, filepath.Base(filepath.Dir(script)))
		}
		repoDeps = notFound
	}

	log.Info("Downloading sources").Send()

	err = getSources(ctx, srcdir, &vars)
	if err != nil {
		return nil, nil, err
	}

	scriptdir := filepath.Dir(script)

	err = setDirVars(ctx, runner, srcdir, pkgdir, scriptdir)
	if err != nil {
		return nil, nil, err
	}

	fn, ok := dec.GetFunc("version")
	if ok {
		log.Info("Executing version()").Send()

		buf := &bytes.Buffer{}

		err = fn(
			ctx,
			interp.Dir(srcdir),
			interp.StdIO(os.Stdin, buf, os.Stderr),
		)
		if err != nil {
			return nil, nil, err
		}

		newVer := strings.TrimSpace(buf.String())
		err = setVersion(ctx, runner, newVer)
		if err != nil {
			return nil, nil, err
		}
		vars.Version = newVer

		log.Info("Updating version").Str("new", newVer).Send()
	}

	fn, ok = dec.GetFunc("prepare")
	if ok {
		log.Info("Executing prepare()").Send()

		err = fn(ctx, interp.Dir(srcdir))
		if err != nil {
			return nil, nil, err
		}
	}

	fn, ok = dec.GetFunc("build")
	if ok {
		log.Info("Executing build()").Send()

		err = fn(ctx, interp.Dir(srcdir))
		if err != nil {
			return nil, nil, err
		}
	}

	fn, ok = dec.GetFunc("package")
	if ok {
		log.Info("Executing package()").Send()

		err = fn(ctx, interp.Dir(srcdir))
		if err != nil {
			return nil, nil, err
		}
	} else {
		log.Fatal("The package() function is required").Send()
	}

	uniq(
		&repoDeps,
		&builtDeps,
		&builtNames,
	)

	pkgInfo := &nfpm.Info{
		Name:        vars.Name,
		Description: vars.Description,
		Arch:        cpu.Arch(),
		Version:     vars.Version,
		Release:     strconv.Itoa(vars.Release),
		Homepage:    vars.Homepage,
		License:     strings.Join(vars.Licenses, ", "),
		Maintainer:  vars.Maintainer,
		Overridables: nfpm.Overridables{
			Conflicts: vars.Conflicts,
			Replaces:  vars.Replaces,
			Provides:  vars.Provides,
			Depends:   append(repoDeps, builtNames...),
		},
	}

	if vars.Epoch != 0 {
		pkgInfo.Epoch = strconv.FormatUint(uint64(vars.Epoch), 10)
	}

	setScripts(&vars, pkgInfo, filepath.Dir(script))

	if slices.Contains(vars.Architectures, "all") {
		pkgInfo.Arch = "all"
	}

	contents := []*files.Content{}
	filepath.Walk(pkgdir, func(path string, fi os.FileInfo, err error) error {
		trimmed := strings.TrimPrefix(path, pkgdir)

		if fi.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}

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

			f.Close()
			return nil
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}

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

		if slices.Contains(vars.Backup, trimmed) {
			fileContent.Type = "config|noreplace"
		}

		contents = append(contents, fileContent)

		return nil
	})

	pkgInfo.Overridables.Contents = contents

	pkgFormat := mgr.Format()
	if format, ok := os.LookupEnv("LURE_PKG_FORMAT"); ok {
		pkgFormat = format
	}

	packager, err := nfpm.Get(pkgFormat)
	if err != nil {
		return nil, nil, err
	}

	pkgName := packager.ConventionalFileName(pkgInfo)
	pkgPath := filepath.Join(baseDir, pkgName)

	pkgPaths := append(builtDeps, pkgPath)
	pkgNames := append(builtNames, vars.Name)

	pkgFile, err := os.Create(pkgPath)
	if err != nil {
		return nil, nil, err
	}

	err = packager.Package(pkgInfo, pkgFile)
	if err != nil {
		return nil, nil, err
	}

	if len(buildDeps) > 0 {
		removeBuildDeps, err := yesNoPrompt("Would you like to remove build dependencies?", false)
		if err != nil {
			return nil, nil, err
		}

		if removeBuildDeps {
			err = mgr.Remove(
				&manager.Opts{
					AsRoot:    true,
					NoConfirm: true,
				},
				buildDeps...,
			)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	uniq(&pkgPaths, &pkgNames)

	return pkgPaths, pkgNames, nil
}

func genBuildEnv(info *distro.OSRelease) []string {
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

	return env
}

func getSources(ctx context.Context, srcdir string, bv *BuildVars) error {
	if len(bv.Sources) != len(bv.Checksums) {
		log.Fatal("The checksums array must be the same length as sources")
	}

	for i, src := range bv.Sources {
		opts := download.GetOptions{
			SourceURL:   src,
			Destination: srcdir,
			EncloseGit:  true,
		}

		if !strings.EqualFold(bv.Checksums[i], "SKIP") {
			checksum, err := hex.DecodeString(bv.Checksums[i])
			if err != nil {
				return err
			}
			opts.SHA256Sum = checksum
		}

		err := download.Get(ctx, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

// setDirVars sets srcdir and pkgdir. It's a very hacky way of doing so,
// but setting the runner's Env and Vars fields doesn't seem to work.
func setDirVars(ctx context.Context, runner *interp.Runner, srcdir, pkgdir, scriptdir string) error {
	cmd := "srcdir='" + srcdir + "'\npkgdir='" + pkgdir + "'\nscriptdir='" + scriptdir + "'\n"
	fl, err := syntax.NewParser().Parse(strings.NewReader(cmd), "vars")
	if err != nil {
		return err
	}
	return runner.Run(ctx, fl)
}

func setScripts(vars *BuildVars, info *nfpm.Info, scriptDir string) {
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

// getBuildVars only gets the build variables, while disabling exec, stat, open, and readdir
func getBuildVars(ctx context.Context, script string, info *distro.OSRelease) (*BuildVars, error) {
	fl, err := os.Open(script)
	if err != nil {
		return nil, err
	}

	file, err := syntax.NewParser().Parse(fl, "lure.sh")
	if err != nil {
		return nil, err
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
		return nil, err
	}

	err = runner.Run(ctx, file)
	if err != nil {
		return nil, err
	}

	dec := decoder.New(info, runner)

	var vars BuildVars
	err = dec.DecodeVars(&vars)
	if err != nil {
		return nil, err
	}

	return &vars, nil
}

// archMatches checks if your system architecture matches
// one of the provided architectures
func archMatches(architectures []string) bool {
	if slices.Contains(architectures, "all") {
		return true
	}

	if slices.Contains(architectures, "arm") {
		architectures = append(architectures, cpu.ARMVariant())
	}

	return slices.Contains(architectures, cpu.Arch())
}

func setVersion(ctx context.Context, r *interp.Runner, to string) error {
	fl, err := syntax.NewParser().Parse(strings.NewReader("version='"+to+"'"), "")
	if err != nil {
		return err
	}
	return r.Run(ctx, fl)
}

// uniq removes all duplicates from string slices
func uniq(ss ...*[]string) {
	for _, s := range ss {
		slices.Sort(*s)
		*s = slices.Compact(*s)
	}
}
