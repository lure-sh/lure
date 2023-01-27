<img src="assets/logo.png" alt="LURE Logo" width="200">

# LURE (Linux User REpository)

[![Go Report Card](https://goreportcard.com/badge/go.arsenm.dev/lure)](https://goreportcard.com/report/go.arsenm.dev/lure)
[![status-badge](https://ci.arsenm.dev/api/badges/Arsen6331/lure/status.svg)](https://ci.arsenm.dev/Arsen6331/lure)
[![linux-user-repository-bin AUR package](https://img.shields.io/aur/version/linux-user-repository-bin?label=lure-bin&logo=archlinux)](https://aur.archlinux.org/packages/lure-bin/)

LURE is a distro-agnostic build system for Linux, similar to the [AUR](https://wiki.archlinux.org/title/Arch_User_Repository). It is currently in an ***alpha*** state and may not be stable. It is currently able to successfully and consistently build and install packages on various distributions, but there are some bugs that still need to be ironed out.

LURE is written in pure Go and has zero dependencies after building. The only things LURE requires are a command for privilege elevation such as `sudo`, `doas`, etc. as well as a supported package manager. Currently, LURE supports `apt`, `pacman`, `apk`, `dnf`, `yum`, and `zypper`. If a supported package manager exists on your system, it will be detected and used automatically.

---

## Installation

### Install script

The LURE install script will automatically download and install the appropriate LURE package on your system. To use it, simply run the following command:

```bash
curl https://www.arsenm.dev/lure.sh | bash
```

**IMPORTANT**: This method is not recommended as it executes any code that is stored at that URL. In order to make sure nothing malicious is going to occur, download the script and inspect it before running.

### Packages

Distro packages and binary archives are provided at the latest Gitea release: https://gitea.arsenm.dev/Arsen6331/lure/releases/latest

LURE is also available on the AUR as [lure-bin](https://aur.archlinux.org/packages/lure-bin)

### Building from source

To build LURE from source, you'll need Go 1.18 or newer. Once Go is installed, clone this repo and run:

```shell
sudo make install
```

---

## Why?

Arch Linux's AUR is a very useful feature. It's one of the main reasons I and many others use Arch on most of their devices. However, Arch is not always a good choice. Whether you're running a server that needs to be stable over long periods of time, or you're a beginner and feel intimidated by Arch, there are many different reasons not to use it. Such useful functionality should not be restricted to only a single distro. That is what LURE is meant to solve.

---

## Documentation

The documentation for LURE is in the [docs](docs) directory in this repo.

---

## Web Interface

LURE now has a web interface! It's open source, licensed under the AGPLv3 (https://gitea.arsenm.dev/Arsen6331/lure-web), and is available at https://lure.arsenm.dev.

---

## Repositories

Unlike the AUR, LURE supports third-party repositories. Also unlike the AUR, LURE's repos are single git repositories containing all the build scripts. Inside each LURE repo, there is a separate directory for each package, containing a `lure.sh` script, which is a PKGBUILD-like build script for LURE. The default repository is hosted on Github: https://github.com/Arsen6331/lure-repo, and information about its packages is displayed at https://lure.arsenm.dev/pkgs.

---

## Dependencies

As mentioned before, LURE has zero dependencies after compilation. Thanks to the following projects for making this possible:

- https://github.com/mvdan/sh
- https://github.com/go-git/go-git
- https://github.com/mholt/archiver
- https://github.com/goreleaser/nfpm
- https://github.com/charmbracelet/bubbletea
- https://gitlab.com/cznic/sqlite

---

## Planned Features

- Automated docker-based testing tool