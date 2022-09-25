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

func (a *APK) Sync() error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apk", "update")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: sync: %w", err)
	}
	return nil
}

func (a *APK) Install(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apk", "add")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: install: %w", err)
	}
	return nil
}

func (a *APK) InstallLocal(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apk", "add", "--allow-untrusted")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: installlocal: %w", err)
	}
	return nil
}

func (a *APK) Remove(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apt", "del")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: remove: %w", err)
	}
	return nil
}

func (a *APK) Upgrade(pkgs ...string) error {
	cmd := exec.Command(getRootCmd(a.rootCmd), "apk", "upgrade")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apk: upgrade: %w", err)
	}
	return nil
}

func (a *APK) UpgradeAll() error {
	return a.Upgrade()
}

func (a *APK) ListInstalled() (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command(getRootCmd(a.rootCmd), "apk", "list", "-I")

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
