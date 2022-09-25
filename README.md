# LURE (Linux User REpository)

LURE is intended to bring the AUR to all distros. It is currently in an ***alpha*** state and may not be stable. It can download a repository, build packages in it using a bash script similar to [PKGBUILD](https://wiki.archlinux.org/title/PKGBUILD), and then install them using your system package manager.

LURE is written in pure Go and has zero dependencies after it's built. The only things LURE needs are a command for privilege elevation such as `sudo`, `doas`, etc. as well as a supported package manager. Currently, LURE supports `apt`, `pacman`, `apk`, `dnf`, `yum`, and `zypper`. If a supported package manager exists on your system, it will be detected and used automatically.