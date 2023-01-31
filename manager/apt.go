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

// APT represents the APT package manager
type APT struct {
	rootCmd string
}

func (*APT) Exists() bool {
	_, err := exec.LookPath("apt")
	return err == nil
}

func (*APT) Name() string {
	return "apt"
}

func (*APT) Format() string {
	return "deb"
}

func (a *APT) SetRootCmd(s string) {
	a.rootCmd = s
}

func (a *APT) Sync(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "update")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: sync: %w", err)
	}
	return nil
}

func (a *APT) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "install")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: install: %w", err)
	}
	return nil
}

func (a *APT) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return a.Install(opts, pkgs...)
}

func (a *APT) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "remove")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: remove: %w", err)
	}
	return nil
}

func (a *APT) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return a.Install(opts, pkgs...)
}

func (a *APT) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "upgrade")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: upgradeall: %w", err)
	}
	return nil
}

func (a *APT) ListInstalled(opts *Opts) (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command("dpkg-query", "-f", "${Package}\u200b${Version}\\n", "-W")

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
		name, version, ok := strings.Cut(scanner.Text(), "\u200b")
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

func (a *APT) getCmd(opts *Opts, mgrCmd string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(a.rootCmd), mgrCmd)
		cmd.Args = append(cmd.Args, opts.Args...)
		cmd.Args = append(cmd.Args, args...)
	} else {
		cmd = exec.Command(mgrCmd, args...)
	}

	if opts.NoConfirm {
		cmd.Args = append(cmd.Args, "-y")
	}

	return cmd
}
