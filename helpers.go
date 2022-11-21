package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.arsenm.dev/lure/internal/shutils"
	"mvdan.cc/sh/v3/interp"
)

var helpers = shutils.ExecFuncs{
	"install-bin":          installHelperCmd("/usr/bin", 0o755),
	"install-systemd-user": installHelperCmd("/usr/lib/systemd/user", 0o644),
	"install-systemd":      installHelperCmd("/usr/lib/systemd/system", 0o644),
	"install-config":       installHelperCmd("/etc", 0o644),
	"install-license":      installHelperCmd("/usr/share/licenses", 0o644),
	"install-manual":       installHelperCmd("/usr/share/man/man1", 0o644),
	"install-desktop":      installHelperCmd("/usr/share/applications", 0o644),
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
