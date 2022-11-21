package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.arsenm.dev/lure/internal/shutils"
	"mvdan.cc/sh/v3/interp"
)

var ErrNoPipe = errors.New("command requires data to be piped in")

var helpers = shutils.ExecFuncs{
	"install-binary":       installHelperCmd("/usr/bin", 0o755),
	"install-systemd-user": installHelperCmd("/usr/lib/systemd/user", 0o644),
	"install-systemd":      installHelperCmd("/usr/lib/systemd/system", 0o644),
	"install-config":       installHelperCmd("/etc", 0o644),
	"install-license":      installHelperCmd("/usr/share/licenses", 0o644),
	"install-manual":       installHelperCmd("/usr/share/man/man1", 0o644),
	"install-desktop":      installHelperCmd("/usr/share/applications", 0o644),
	"install-completion":   installCompletionCmd,
}

func installHelperCmd(prefix string, perms os.FileMode) shutils.ExecFunc {
	return func(hc interp.HandlerContext, cmd string, args []string) error {
		if len(args) < 1 {
			return shutils.InsufficientArgsError(cmd, 1, len(args))
		}

		from := args[0]
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
		prefix = "/usr/share/bash-completion/completion"
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
