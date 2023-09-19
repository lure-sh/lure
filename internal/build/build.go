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

	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"
	"go.elara.ws/logger/log"
	"go.elara.ws/lure/distro"
	"go.elara.ws/lure/internal/cliutils"
	"go.elara.ws/lure/internal/config"
	"go.elara.ws/lure/internal/cpu"
	"go.elara.ws/lure/internal/db"
	"go.elara.ws/lure/internal/dl"
	"go.elara.ws/lure/internal/repos"
	"go.elara.ws/lure/internal/shutils"
	"go.elara.ws/lure/internal/shutils/decoder"
	"go.elara.ws/lure/internal/shutils/helpers"
	"go.elara.ws/lure/internal/types"
	"go.elara.ws/lure/manager"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

// BuildPackage builds the script at the given path. It returns two slices. One contains the paths
// to the built package(s), the other contains the names of the built package(s).
func BuildPackage(ctx context.Context, opts types.BuildOpts) ([]string, []string, error) {
	info, err := distro.ParseOSRelease(ctx)
	if err != nil {
		return nil, nil, err
	}

	fl, err := parseScript(info, opts.Script)
	if err != nil {
		return nil, nil, err
	}

	vars, err := executeFirstPass(ctx, info, fl, opts.Script)
	if err != nil {
		return nil, nil, err
	}

	dirs := getDirs(vars, opts.Script)

	if !opts.Clean {
		builtPkgPath, ok, err := checkForBuiltPackage(opts.Manager, vars, getPkgFormat(opts.Manager), dirs.BaseDir)
		if err != nil {
			return nil, nil, err
		}

		if ok {
			return []string{builtPkgPath}, nil, err
		}
	}

	err = cliutils.PromptViewScript(opts.Script, vars.Name, config.Config().PagerStyle, opts.Interactive)
	if err != nil {
		log.Fatal("Failed to prompt user to view build script").Err(err).Send()
	}

	log.Info("Building package").Str("name", vars.Name).Str("version", vars.Version).Send()

	dec, err := executeSecondPass(ctx, info, fl, dirs)
	if err != nil {
		return nil, nil, err
	}

	installed, err := opts.Manager.ListInstalled(nil)
	if err != nil {
		return nil, nil, err
	}

	cont, err := performChecks(vars, opts.Interactive, installed)
	if err != nil {
		return nil, nil, err
	} else if !cont {
		os.Exit(1)
	}

	err = prepareDirs(dirs)
	if err != nil {
		return nil, nil, err
	}

	buildDeps, err := installBuildDeps(ctx, vars, opts, installed)
	if err != nil {
		return nil, nil, err
	}

	builtPaths, builtNames, repoDeps, err := installDeps(ctx, opts, vars)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Downloading sources").Send()

	err = getSources(ctx, dirs.SrcDir, vars)
	if err != nil {
		return nil, nil, err
	}

	err = executeFunctions(ctx, dec, dirs, vars)
	if err != nil {
		return nil, nil, err
	}

	log.Info("Building package metadata").Str("name", vars.Name).Send()

	pkgInfo, err := buildPkgMetadata(vars, dirs, append(repoDeps, builtNames...))
	if err != nil {
		return nil, nil, err
	}

	packager, err := nfpm.Get(getPkgFormat(opts.Manager))
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

	err = removeBuildDeps(buildDeps, opts)
	if err != nil {
		return nil, nil, err
	}

	// Add the path and name of the package we just built to the
	// appropriate slices
	pkgPaths := append(builtPaths, pkgPath)
	pkgNames := append(builtNames, vars.Name)

	pkgPaths = removeDuplicates(pkgPaths)
	pkgNames = removeDuplicates(pkgNames)

	return pkgPaths, pkgNames, nil
}

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

func executeFirstPass(ctx context.Context, info *distro.OSRelease, fl *syntax.File, script string) (*types.BuildVars, error) {
	scriptDir := filepath.Dir(script)
	env := createBuildEnvVars(info, types.Directories{ScriptDir: scriptDir})

	// The first pass is just used to get variable values and runs before
	// the script is displayed, so it is restricted so as to prevent malicious
	// code from executing.
	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(helpers.Restricted.ExecHandler(shutils.NopExec)),
		interp.ReadDirHandler(shutils.RestrictedReadDir(scriptDir)),
		interp.StatHandler(shutils.RestrictedStat(scriptDir)),
		interp.OpenHandler(shutils.RestrictedOpen(scriptDir)),
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

func getDirs(vars *types.BuildVars, script string) types.Directories {
	baseDir := filepath.Join(config.GetPaths().PkgsDir, vars.Name)
	return types.Directories{
		BaseDir:   baseDir,
		SrcDir:    filepath.Join(baseDir, "src"),
		PkgDir:    filepath.Join(baseDir, "pkg"),
		ScriptDir: filepath.Dir(script),
	}
}

func executeSecondPass(ctx context.Context, info *distro.OSRelease, fl *syntax.File, dirs types.Directories) (*decoder.Decoder, error) {
	env := createBuildEnvVars(info, dirs)
	// The second pass will be used to execute the actual functions,
	// so it cannot be restricted. The script has already been displayed
	// to the user by this point, so it should be safe
	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),
		interp.ExecHandler(helpers.Helpers.ExecHandler(nil)),
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

func performChecks(vars *types.BuildVars, interactive bool, installed map[string]string) (bool, error) {
	if !cpu.IsCompatibleWith(cpu.Arch(), vars.Architectures) {
		cont, err := cliutils.YesNoPrompt("Your system's CPU architecture doesn't match this package. Do you want to build anyway?", interactive, true)
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

func installBuildDeps(ctx context.Context, vars *types.BuildVars, opts types.BuildOpts, installed map[string]string) ([]string, error) {
	var buildDeps []string
	if len(vars.BuildDepends) > 0 {
		found, notFound, err := repos.FindPkgs(vars.BuildDepends)
		if err != nil {
			return nil, err
		}

		found = filterBuildDeps(found, installed)

		log.Info("Installing build dependencies").Send()

		flattened := cliutils.FlattenPkgs(found, "install", opts.Interactive)
		buildDeps = packageNames(flattened)
		InstallPkgs(ctx, flattened, notFound, opts)
	}
	return buildDeps, nil
}

func installDeps(ctx context.Context, opts types.BuildOpts, vars *types.BuildVars) (builtPaths, builtNames, repoDeps []string, err error) {
	if len(vars.Depends) > 0 {
		log.Info("Installing dependencies").Send()

		found, notFound, err := repos.FindPkgs(vars.Depends)
		if err != nil {
			return nil, nil, nil, err
		}
		repoDeps = notFound

		// If there are multiple options for some packages, flatten them all into a single slice
		pkgs := cliutils.FlattenPkgs(found, "install", opts.Interactive)
		scripts := GetScriptPaths(pkgs)
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

	repoDeps = removeDuplicates(repoDeps)
	builtPaths = removeDuplicates(builtPaths)
	builtNames = removeDuplicates(builtNames)
	return builtPaths, builtNames, repoDeps, nil
}

func executeFunctions(ctx context.Context, dec *decoder.Decoder, dirs types.Directories, vars *types.BuildVars) (err error) {
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

func buildPkgMetadata(vars *types.BuildVars, dirs types.Directories, deps []string) (*nfpm.Info, error) {
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

func buildContents(vars *types.BuildVars, dirs types.Directories) ([]*files.Content, error) {
	contents := []*files.Content{}
	err := filepath.Walk(dirs.PkgDir, func(path string, fi os.FileInfo, err error) error {
		trimmed := strings.TrimPrefix(path, dirs.PkgDir)

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

		if slices.Contains(vars.Backup, trimmed) {
			fileContent.Type = "config|noreplace"
		}

		contents = append(contents, fileContent)

		return nil
	})
	return contents, err
}

func removeBuildDeps(buildDeps []string, opts types.BuildOpts) error {
	if len(buildDeps) > 0 {
		removeBuildDeps, err := cliutils.YesNoPrompt("Would you like to remove the build dependencies?", opts.Interactive, false)
		if err != nil {
			return err
		}

		if removeBuildDeps {
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

func getPkgFormat(mgr manager.Manager) string {
	pkgFormat := mgr.Format()
	if format, ok := os.LookupEnv("LURE_PKG_FORMAT"); ok {
		pkgFormat = format
	}
	return pkgFormat
}

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

func getSources(ctx context.Context, srcdir string, bv *types.BuildVars) error {
	if len(bv.Sources) != len(bv.Checksums) {
		log.Fatal("The checksums array must be the same length as sources").Send()
	}

	for i, src := range bv.Sources {
		opts := dl.Options{
			Name:        fmt.Sprintf("%s[%d]", bv.Name, i),
			URL:         src,
			Destination: srcdir,
			Progress:    os.Stderr,
		}

		if !strings.EqualFold(bv.Checksums[i], "SKIP") {
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

func setVersion(ctx context.Context, r *interp.Runner, to string) error {
	fl, err := syntax.NewParser().Parse(strings.NewReader("version='"+to+"'"), "")
	if err != nil {
		return err
	}
	return r.Run(ctx, fl)
}

// filterBuildDeps returns a map without any dependencies that are already installed
func filterBuildDeps(found map[string][]db.Package, installed map[string]string) map[string][]db.Package {
	out := map[string][]db.Package{}
	for name, pkgs := range found {
		var inner []db.Package
		for _, pkg := range pkgs {
			if _, ok := installed[pkg.Name]; !ok {
				addToFiltered := true
				for _, provides := range pkg.Provides.Val {
					if _, ok := installed[provides]; ok {
						addToFiltered = false
						break
					}
				}

				if addToFiltered {
					inner = append(inner, pkg)
				}
			}
		}

		if len(inner) > 0 {
			out[name] = inner
		}
	}
	return out
}

func packageNames(pkgs []db.Package) []string {
	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = p.Name
	}
	return names
}

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
