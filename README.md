<img src="assets/logo.png" alt="LURE Logo" width="200">

# LURE (Linux User REpository)

[![Go Report Card](https://goreportcard.com/badge/go.elara.ws/lure)](https://goreportcard.com/report/go.elara.ws/lure)
[![status-badge](https://ci.elara.ws/api/badges/lure/lure/status.svg)](https://ci.elara.ws/lure/lure)
[![linux-user-repository-bin AUR package](https://img.shields.io/aur/version/linux-user-repository-bin?label=linux-user-repository-bin&logo=archlinux)](https://aur.archlinux.org/packages/linux-user-repository-bin/)

LURE is a distro-agnostic build system for Linux, similar to the [AUR](https://wiki.archlinux.org/title/Arch_User_Repository). It is currently in **beta**. Most major bugs have been fixed, and most major features have been added. LURE is ready for general use, but may still break or change occasionally.

LURE is written in pure Go and has zero dependencies after building. The only things LURE requires are a command for privilege elevation such as `sudo`, `doas`, etc. as well as a supported package manager. Currently, LURE supports `apt`, `pacman`, `apk`, `dnf`, `yum`, and `zypper`. If a supported package manager exists on your system, it will be detected and used automatically.

---

## Installation

### Install script

The LURE install script will automatically download and install the appropriate LURE package on your system. To use it, simply run the following command:

```bash
curl -fsSL lure.sh/install | bash
```

**IMPORTANT**: This method is not recommended as it executes any code that is stored at that URL. In order to make sure nothing malicious is going to occur, download the script and inspect it before running.

### Packages

Distro packages and binary archives are provided at the latest Gitea release: https://gitea.elara.ws/lure/lure/releases/latest

LURE is also available on the AUR as [linux-user-repository-bin](https://aur.archlinux.org/packages/linux-user-repository-bin)

### Building from source

To build LURE from source, you'll need Go 1.18 or newer. Once Go is installed, clone this repo and run:

```shell
sudo make install
```

---

## Why?

LURE was created because packaging software for multiple Linux distros can be difficult and error-prone, and installing those packages can be a nightmare for users unless they're available in their distro's official repositories. It automates the process of building and installing unofficial packages.

---

## Documentation

The documentation for LURE is in the [docs](docs) directory in this repo.

---

## Web Interface

LURE has an open source web interface, licensed under the AGPLv3 (https://gitea.elara.ws/lure/lure-web), and it's available at https://lure.sh/.

---

## Repositories

LURE's repos are git repositories that contain a directory for each package, with a `lure.sh` file inside. The `lure.sh` file tells LURE how to build the package and information about it. `lure.sh` scripts are similar to the AUR's PKGBUILD scripts.

---

## Acknowledgements

Thanks to the following projects for making LURE possible:

- https://github.com/mvdan/sh
- https://github.com/go-git/go-git
- https://github.com/mholt/archiver
- https://github.com/goreleaser/nfpm
- https://github.com/charmbracelet/bubbletea
- https://gitlab.com/cznic/sqlite