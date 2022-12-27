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

package manager

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// Zypper represents the Zypper package manager
type Zypper struct {
	rootCmd string
}

func (*Zypper) Exists() bool {
	_, err := exec.LookPath("zypper")
	return err == nil
}

func (*Zypper) Name() string {
	return "zypper"
}

func (*Zypper) Format() string {
	return "rpm"
}

func (z *Zypper) SetRootCmd(s string) {
	z.rootCmd = s
}

func (z *Zypper) Sync(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "refresh")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: sync: %w", err)
	}
	return nil
}

func (z *Zypper) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "install", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: install: %w", err)
	}
	return nil
}

func (z *Zypper) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return z.Install(opts, pkgs...)
}

func (z *Zypper) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "remove", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: remove: %w", err)
	}
	return nil
}

func (z *Zypper) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "update", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: upgrade: %w", err)
	}
	return nil
}

func (z *Zypper) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "update", "-y")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: upgradeall: %w", err)
	}
	return nil
}

func (z *Zypper) ListInstalled(opts *Opts) (map[string]string, error) {
	opts = ensureOpts(opts)
	out := map[string]string{}

	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(z.rootCmd), "rpm", "-qa", "--queryformat", "%{NAME}\u200b%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\n")
	} else {
		cmd = exec.Command("rpm", "-qa", "--queryformat", "%{NAME}\u200b%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\n")
	}

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
		version = strings.TrimPrefix(version, "0:")
		out[name] = version
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (z *Zypper) getCmd(opts *Opts, mgrCmd string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(z.rootCmd), mgrCmd)
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
