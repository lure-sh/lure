/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Arsen Musayelyan
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

package manager

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// APK represents the APK package manager
type APK struct {
	rootCmd string
}

func (*APK) Exists() bool {
	_, err := exec.LookPath("apk")
	return err == nil
}

func (*APK) Name() string {
	return "apk"
}

func (*APK) Format() string {
	return "apk"
}

func (a *APK) SetRootCmd(s string) {
	a.rootCmd = s
}

func (a *APK) Sync(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apk", "update")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: sync: %w", err)
	}
	return nil
}

func (a *APK) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apk", "add")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: install: %w", err)
	}
	return nil
}

func (a *APK) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apk", "add", "--allow-untrusted")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: installlocal: %w", err)
	}
	return nil
}

func (a *APK) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apk", "del")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: remove: %w", err)
	}
	return nil
}

func (a *APK) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apk", "upgrade")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: upgrade: %w", err)
	}
	return nil
}

func (a *APK) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	return a.Upgrade(opts)
}

func (a *APK) ListInstalled(opts *Opts) (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command("apk", "list", "-I")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		name, info, ok := strings.Cut(scanner.Text(), "-")
		if !ok {
			continue
		}

		version, _, ok := strings.Cut(info, " ")
		if !ok {
			continue
		}

		out[name] = version
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *APK) getCmd(opts *Opts, mgrCmd string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(a.rootCmd), mgrCmd)
		cmd.Args = append(cmd.Args, opts.Args...)
		cmd.Args = append(cmd.Args, args...)
	} else {
		cmd = exec.Command(mgrCmd, args...)
	}

	if !opts.NoConfirm {
		cmd.Args = append(cmd.Args, "-i")
	}

	return cmd
}
