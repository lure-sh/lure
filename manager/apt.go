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

func (a *APT) Sync() error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apt", "update", "-y")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: sync: %w", err)
	}
	return nil
}

func (a *APT) Install(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apt", "install", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: install: %w", err)
	}
	return nil
}

func (a *APT) InstallLocal(pkgs ...string) error {
	return a.Install(pkgs...)
}

func (a *APT) Remove(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apt", "remove", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: remove: %w", err)
	}
	return nil
}

func (a *APT) Upgrade(pkgs ...string) error {
	return a.Install(pkgs...)
}

func (a *APT) UpgradeAll() error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apt", "upgrade", "-y")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: upgradeall: %w", err)
	}
	return nil
}

func (a *APT) ListInstalled() (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command(getRootCmd(a.rootCmd), "dpkg-query", "-f", "${Package}\u200b${Version}\\n", "-W")

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
