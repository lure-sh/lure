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

func (z *Zypper) Sync() error {
	cmd := exec.Command(getRootCmd(z.rootCmd), "zypper", "refresh")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: sync: %w", err)
	}
	return nil
}

func (z *Zypper) Install(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(z.rootCmd), "zypper", "install", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: install: %w", err)
	}
	return nil
}

func (z *Zypper) InstallLocal(pkgs ...string) error {
	return z.Install(pkgs...)
}

func (z *Zypper) Remove(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(z.rootCmd), "zypper", "remove", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: remove: %w", err)
	}
	return nil
}

func (z *Zypper) Upgrade(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(z.rootCmd), "zypper", "update", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: upgrade: %w", err)
	}
	return nil
}

func (z *Zypper) UpgradeAll() error {
	cmd := exec.Command(getRootCmd(z.rootCmd), "zypper", "update", "-y")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: upgradeall: %w", err)
	}
	return nil
}

func (z *Zypper) ListInstalled() (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command(getRootCmd(z.rootCmd), "rpm", "-qa", "--queryformat", "%{NAME}\u200b%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\n")

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
