# LURE (Linux User REpository)

[![Go Report Card](https://goreportcard.com/badge/go.arsenm.dev/lure)](https://goreportcard.com/report/go.arsenm.dev/lure)
[![lure-bin AUR package](https://img.shields.io/aur/version/lure-bin?label=itd-bin&logo=archlinux)](https://aur.archlinux.org/packages/lure-bin/)

LURE is intended to bring the AUR to all distros. It is currently in an ***alpha*** state and may not be stable. It can download a repository, build packages in it using a bash script similar to [PKGBUILD](https://wiki.archlinux.org/title/PKGBUILD), and then install them using your system package manager.

LURE is written in pure Go and has zero dependencies after it's built. The only things LURE needs are a command for privilege elevation such as `sudo`, `doas`, etc. as well as a supported package manager. Currently, LURE supports `apt`, `pacman`, `apk`, `dnf`, `yum`, and `zypper`. If a supported package manager exists on your system, it will be detected and used automatically.

---

## Installation

Distro packages and binary archives are provided at the latest Gitea release: https://gitea.arsenm.dev/Arsen6331/lure/releases/latest

LURE is also available on the AUR as [lure-bin](https://aur.archlinux.org/packages/lure-bin)

### Building from source

To build LURE from source, you'll need Go 1.18 or newer. Once Go is installed, clone this repo and run:

```shell
sudo make install
```

---

## Why?

The AUR is an amazing feature, and it's one of the main reasons I use Arch on all my daily driver devices. It is really simple while providing really useful functionality. I feel such a solution shouldn't be stuck in only a single distro, so I made LURE.

Like the AUR, it uses simple bash build scripts, but it doesn't depend on bash being installed at all. It uses an embedded, pure Go implementation of bash instead. Similarly, it uses Git to download the repos and sources, but doesn't depend on Git being installed.

This means it's really easy to deploy LURE on any distro that it has support for and on any CPU architecture. It also supports and automatically detects many package managers, so it's not limited to just `pacman`.

---

## Distro Overrides

Allowing LURE to run on different distros provides some challenges. For example, some distros use different names for their packages. This is solved using distro overrides. Any variable or function used in a LURE build script may be overridden based on distro and CPU architecture. The way you do this is by appending the distro and/or architecture to the end of the name. For example, [ITD](https://gitea.arsenm.dev/Arsen6331/itd) depends on the `pactl` command as well as DBus and BlueZ. These are named somewhat differently on different distros. For ITD, I use the following for the dependencies:

```bash
deps=('dbus' 'bluez' 'pulseaudio-utils')
deps_arch=('dbus' 'bluez' 'libpulse')
deps_opensuse=('dbus-1' 'bluez' 'pulseaudio-utils')
```

Appending `arch` and `opensuse` to the end causes LURE to use the appropriate array based on the distro. If on Arch Linux, it will use `deps_arch`. If on OpenSUSE, it will use `deps_opensuse`, and if on anything else, it will use `deps`.

Names are checked in the following order:

- $name_$architecture_$distro
- $name_$distro
- $name_$architecture
- $name

Distro detection is performed by reading the `/usr/lib/os-release` and `/etc/os-release` files.

---

## Repositories

Unlike the AUR, LURE supports using multiple repos. Also unlike the AUR, LURE's repos are a single git repo containing all the build scripts. Inside each LURE repo, there should be a separate directory for each package containing a `lure.sh` script, which is a PKGBUILD-like build script for LURE. The default repository is hosted on Github: https://github.com/Arsen6331/lure-repo.

---

## Dependencies

As mentioned before, LURE has zero dependencies after it's built. All functionality that could be pure Go is pure Go. Thanks to the following packages for making this possible:

- Bash: https://github.com/mvdan/sh
- Git: https://github.com/go-git/go-git
- Archiver: https://github.com/mholt/archiver
- nfpm: https://github.com/goreleaser/nfpm

---

## Planned Features

- Source patching via .patch files
- Binary releases using goreleaser + CI
- Automated install script
- Automated docker-based testing tool
- Web interface for repos