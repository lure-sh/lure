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

// Pacman represents the Pacman package manager
type Pacman struct {
	rootCmd string
}

func (*Pacman) Exists() bool {
	_, err := exec.LookPath("pacman")
	return err == nil
}

func (*Pacman) Name() string {
	return "pacman"
}

func (*Pacman) Format() string {
	return "archlinux"
}

func (p *Pacman) SetRootCmd(s string) {
	p.rootCmd = s
}

func (p *Pacman) Sync() error {
	cmd := exec.Command(getRootCmd(p.rootCmd), "pacman", "--noconfirm", "-Sy")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pacman: sync: %w", err)
	}
	return nil
}

func (p *Pacman) Install(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(p.rootCmd), "pacman", "--noconfirm", "-S")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pacman: install: %w", err)
	}
	return nil
}

func (p *Pacman) InstallLocal(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(p.rootCmd), "pacman", "--noconfirm", "-U")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pacman: installlocal: %w", err)
	}
	return nil
}

func (p *Pacman) Remove(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(p.rootCmd), "pacman", "--noconfirm", "-R")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pacman: remove: %w", err)
	}
	return nil
}

func (p *Pacman) Upgrade(pkgs ...string) error {
	return p.Install(pkgs...)
}

func (p *Pacman) UpgradeAll() error {
	cmd := exec.Command(getRootCmd(p.rootCmd), "pacman", "--noconfirm", "-Su")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pacman: upgradeall: %w", err)
	}
	return nil
}

func (p *Pacman) ListInstalled() (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command(getRootCmd(p.rootCmd), "pacman", "-Q")

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
		name, version, ok := strings.Cut(scanner.Text(), " ")
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
