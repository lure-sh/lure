package manager

import (
	"os"
	"os/exec"
)

// DefaultRootCmd is the command used for privilege elevation by default
var DefaultRootCmd = "sudo"

var managers = []Manager{
	&Pacman{},
	&APT{},
	&DNF{},
	&YUM{},
	&APK{},
	&Zypper{},
}

// Register registers a new package manager
func Register(m Manager) {
	managers = append(managers, m)
}

// Manager represents a system package manager
type Manager interface {
	// Name returns the name of the manager.
	Name() string
	// Format returns the packaging format of the manager.
	// 	Examples: rpm, deb, apk
	Format() string
	// Returns true if the package manager exists on the system.
	Exists() bool
	// Sets the command used to elevate privileges. Defaults to DefaultRootCmd.
	SetRootCmd(string)
	// Sync fetches repositories without installing anything
	Sync() error
	// Install installs packages
	Install(...string) error
	// Remove uninstalls packages
	Remove(...string) error
	// Upgrade upgrades packages
	Upgrade(...string) error
	// InstallLocal installs packages from local files rather than repos
	InstallLocal(...string) error
	// UpgradeAll upgrades all packages
	UpgradeAll() error
	// ListInstalled returns all installed packages mapped to their versions
	ListInstalled() (map[string]string, error)
}

// Detect returns the package manager detected on the system
func Detect() Manager {
	for _, mgr := range managers {
		if mgr.Exists() {
			return mgr
		}
	}
	return nil
}

// Get returns the package manager with the given name
func Get(name string) Manager {
	for _, mgr := range managers {
		if mgr.Name() == name {
			return mgr
		}
	}
	return nil
}

// getRootCmd returns rootCmd if it's not empty, otherwise returns DefaultRootCmd
func getRootCmd(rootCmd string) string {
	if rootCmd != "" {
		return rootCmd
	}
	return DefaultRootCmd
}

func setCmdEnv(cmd *exec.Cmd) {
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}
