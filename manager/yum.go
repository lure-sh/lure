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

// YUM represents the YUM package manager
type YUM struct {
	rootCmd string
}

func (*YUM) Exists() bool {
	_, err := exec.LookPath("yum")
	return err == nil
}

func (*YUM) Name() string {
	return "yum"
}

func (*YUM) Format() string {
	return "rpm"
}

func (y *YUM) SetRootCmd(s string) {
	y.rootCmd = s
}

func (y *YUM) Sync(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := y.getCmd(opts, "yum", "upgrade")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yum: sync: %w", err)
	}
	return nil
}

func (y *YUM) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := y.getCmd(opts, "yum", "install", "--allowerasing")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yum: install: %w", err)
	}
	return nil
}

func (y *YUM) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return y.Install(opts, pkgs...)
}

func (y *YUM) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := y.getCmd(opts, "yum", "remove")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yum: remove: %w", err)
	}
	return nil
}

func (y *YUM) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := y.getCmd(opts, "yum", "upgrade")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yum: upgrade: %w", err)
	}
	return nil
}

func (y *YUM) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := y.getCmd(opts, "yum", "upgrade")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yum: upgradeall: %w", err)
	}
	return nil
}

func (y *YUM) ListInstalled(opts *Opts) (map[string]string, error) {
	opts = ensureOpts(opts)
	out := map[string]string{}

	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(y.rootCmd), "rpm", "-qa", "--queryformat", "%{NAME}\u200b%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\n")
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

func (y *YUM) getCmd(opts *Opts, mgrCmd string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(y.rootCmd), mgrCmd)
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
