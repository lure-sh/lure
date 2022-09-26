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

// DNF represents the DNF package manager
type DNF struct {
	rootCmd string
}

func (*DNF) Exists() bool {
	_, err := exec.LookPath("dnf")
	return err == nil
}

func (*DNF) Name() string {
	return "dnf"
}

func (*DNF) Format() string {
	return "rpm"
}

func (d *DNF) SetRootCmd(s string) {
	d.rootCmd = s
}

func (d *DNF) Sync() error {
	cmd := exec.Command(getRootCmd(d.rootCmd), "dnf", "upgrade", "--assumeno")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: sync: %w", err)
	}
	return nil
}

func (d *DNF) Install(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(d.rootCmd), "dnf", "install", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: install: %w", err)
	}
	return nil
}

func (d *DNF) InstallLocal(pkgs ...string) error {
	return d.Install(pkgs...)
}

func (d *DNF) Remove(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(d.rootCmd), "dnf", "remove", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: remove: %w", err)
	}
	return nil
}

func (d *DNF) Upgrade(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(d.rootCmd), "dnf", "upgrade", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: upgrade: %w", err)
	}
	return nil
}

func (d *DNF) UpgradeAll() error {
	cmd := exec.Command(getRootCmd(d.rootCmd), "dnf", "upgrade", "-y")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: upgradeall: %w", err)
	}
	return nil
}

func (d *DNF) ListInstalled() (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command(getRootCmd(d.rootCmd), "rpm", "-qa", "--queryformat", "%{NAME}\u200b%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\n")

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
