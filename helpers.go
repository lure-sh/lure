package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"go.arsenm.dev/lure/internal/shutils"
	"golang.org/x/exp/slices"
	"mvdan.cc/sh/v3/interp"
)

var (
	ErrNoPipe         = errors.New("command requires data to be piped in")
	ErrNoDetectManNum = errors.New("manual number cannot be detected from the filename")
)

var helpers = shutils.ExecFuncs{
	"install-binary":       installHelperCmd("/usr/bin", 0o755),
	"install-systemd-user": installHelperCmd("/usr/lib/systemd/user", 0o644),
	"install-systemd":      installHelperCmd("/usr/lib/systemd/system", 0o644),
	"install-config":       installHelperCmd("/etc", 0o644),
	"install-license":      installHelperCmd("/usr/share/licenses", 0o644),
	"install-desktop":      installHelperCmd("/usr/share/applications", 0o644),
	"install-manual":       installManualCmd,
	"install-completion":   installCompletionCmd,
	"install-library":      installLibraryCmd,
	"git-version":          gitVersionCmd,
}

func installHelperCmd(prefix string, perms os.FileMode) shutils.ExecFunc {
	return func(hc interp.HandlerContext, cmd string, args []string) error {
		if len(args) < 1 {
			return shutils.InsufficientArgsError(cmd, 1, len(args))
		}

		from := resolvePath(hc, args[0])
		to := ""
		if len(args) > 1 {
			to = filepath.Join(hc.Env.Get("pkgdir").Str, prefix, args[1])
		} else {
			to = filepath.Join(hc.Env.Get("pkgdir").Str, prefix, filepath.Base(from))
		}

		err := helperInstall(from, to, perms)
		if err != nil {
			return fmt.Errorf("%s: %w", cmd, err)
		}
		return nil
	}
}

func installManualCmd(hc interp.HandlerContext, cmd string, args []string) error {
	if len(args) < 1 {
		return shutils.InsufficientArgsError(cmd, 1, len(args))
	}

	from := resolvePath(hc, args[0])
	number := filepath.Base(from)
	// The man page may be compressed with gzip.
	// If it is, the .gz extension must be removed to properly
	// detect the number at the end of the filename.
	number = strings.TrimSuffix(number, ".gz")
	number = strings.TrimPrefix(filepath.Ext(number), ".")

	// If number is not actually a number, return an error
	if _, err := strconv.Atoi(number); err != nil {
		return fmt.Errorf("install-manual: %w", ErrNoDetectManNum)
	}

	prefix := "/usr/share/man/man" + number
	to := filepath.Join(hc.Env.Get("pkgdir").Str, prefix, filepath.Base(from))

	return helperInstall(from, to, 0o644)
}

func installCompletionCmd(hc interp.HandlerContext, cmd string, args []string) error {
	// If the command's stdin is the same as the system's,
	// that means nothing was piped in. In this case, return an error.
	if hc.Stdin == os.Stdin {
		return fmt.Errorf("install-completion: %w", ErrNoPipe)
	}

	if len(args) < 2 {
		return shutils.InsufficientArgsError(cmd, 2, len(args))
	}

	shell := args[0]
	name := args[1]

	var prefix string
	switch shell {
	case "bash":
		prefix = "/usr/share/bash-completion/completions"
	case "zsh":
		prefix = "/usr/share/zsh/site-functions"
		name = "_" + name
	case "fish":
		prefix = "/usr/share/fish/vendor_completions.d"
		name += ".fish"
	}

	path := filepath.Join(hc.Env.Get("pkgdir").Str, prefix, name)

	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return err
	}

	dst, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, hc.Stdin)
	return err
}

func installLibraryCmd(hc interp.HandlerContext, cmd string, args []string) error {
	prefix := getLibPrefix(hc)
	fn := installHelperCmd(prefix, 0o755)
	return fn(hc, cmd, args)
}

// See https://wiki.debian.org/Multiarch/Tuples
var multiarchTupleMap = map[string]string{
	"386":      "i386-linux-gnu",
	"amd64":    "x86_64-linux-gnu",
	"arm5":     "arm-linux-gnueabi",
	"arm6":     "arm-linux-gnueabihf",
	"arm7":     "arm-linux-gnueabihf",
	"arm64":    "aarch64-linux-gnu",
	"mips":     "mips-linux-gnu",
	"mipsle":   "mipsel-linux-gnu",
	"mips64":   "mips64-linux-gnuabi64",
	"mips64le": "mips64el-linux-gnuabi64",
	"ppc64":    "powerpc64-linux-gnu",
	"ppc64le":  "powerpc64le-linux-gnu",
	"s390x":    "s390x-linux-gnu",
	"riscv64":  "riscv64-linux-gnu",
	"loong64":  "loongarch64-linux-gnu",
}

// usrLibDistros is a list of distros that don't support
// /usr/lib64, and must use /usr/lib
var usrLibDistros = []string{
	"arch",
	"alpine",
	"void",
	"chimera",
}

// Based on CMake's GNUInstallDirs
func getLibPrefix(hc interp.HandlerContext) string {
	if dir, ok := os.LookupEnv("LURE_LIB_DIR"); ok {
		return dir
	}

	out := "/usr/lib"

	distroID := hc.Env.Get("DISTRO_ID").Str
	distroLike := strings.Split(hc.Env.Get("DISTRO_ID_LIKE").Str, " ")

	for _, usrLibDistro := range usrLibDistros {
		if distroID == usrLibDistro || slices.Contains(distroLike, usrLibDistro) {
			return out
		}
	}

	wordSize := unsafe.Sizeof(uintptr(0))
	if wordSize == 8 {
		out = "/usr/lib64"
	}

	architecture := hc.Env.Get("ARCH").Str

	if distroID == "debian" || slices.Contains(distroLike, "debian") {
		triple, ok := multiarchTupleMap[architecture]
		if ok {
			out = filepath.Join("/usr/lib", triple)
		}
	}

	return out
}

func gitVersionCmd(hc interp.HandlerContext, cmd string, args []string) error {
	path := hc.Dir
	if len(args) > 0 {
		path = resolvePath(hc, args[0])
	}

	r, err := git.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("git-version: %w", err)
	}

	revNum := 0
	commits, err := r.Log(&git.LogOptions{})
	if err != nil {
		return fmt.Errorf("git-version: %w", err)
	}

	commits.ForEach(func(*object.Commit) error {
		revNum++
		return nil
	})

	HEAD, err := r.Head()
	if err != nil {
		return fmt.Errorf("git-version: %w", err)
	}

	hash := HEAD.Hash().String()

	fmt.Fprintf(hc.Stdout, "%d.%s", revNum, hash[:7])

	return nil
}

func helperInstall(from, to string, perms os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(to), 0o755)
	if err != nil {
		return err
	}

	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(to, os.O_TRUNC|os.O_CREATE|os.O_RDWR, perms)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func resolvePath(hc interp.HandlerContext, path string) string {
	if !filepath.IsAbs(path) {
		return filepath.Join(hc.Dir, path)
	}
	return path
}
