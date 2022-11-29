# LURE Build Scripts

LURE uses build scripts similar to the AUR's PKGBUILDs. This is the documentation for those scripts.

---

## Table of Contents

- [Distro Overrides](#distro-overrides)
- [Variables](#variables)
    - [name](#name)
    - [version](#version)
    - [release](#release)
    - [epoch](#epoch)
    - [desc](#desc)
    - [homepage](#homepage)
    - [maintainer](#maintainer)
    - [architectures](#architectures)
    - [licenses](#licenses)
    - [provides](#provides)
    - [conflicts](#conflicts)
    - [deps](#deps)
    - [build_deps](#build_deps)
    - [replaces](#replaces)
    - [sources](#sources)
    - [checksums](#checksums)
    - [backup](#backup)
    - [scripts](#scripts)
- [Functions](#functions)
    - [prepare](#prepare)
    - [version](#version-1)
    - [build](#build)
    - [package](#package)
- [Environment Variables](#environment-variables)
    - [DISTRO_NAME](#distro_name)
    - [DISTRO_PRETTY_NAME](#distro_pretty_name)
    - [DISTRO_ID](#distro_id)
    - [DISTRO_VERSION_ID](#distro_version_id)
    - [ARCH](#arch)
    - [NCPU](#ncpu)
- [Helper Commands](#helper-commands)
    - [install-binary](#install-binary)
    - [install-systemd](#install-systemd)
    - [install-systemd-user](#install-systemd-user)
    - [install-config](#install-config)
    - [install-license](#install-license)
    - [install-completion](#install-completion)
    - [install-manual](#install-manual)
    - [install-desktop](#install-desktop)
    - [install-library](#install-library)
    - [git-version](#git-version)

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

### Like distros

Inside the `os-release` file, there is a list of "like" distros. LURE takes this into account. For example, if a script contains `deps_debian` but not `deps_ubuntu`, Ubuntu builds will use `deps_debian` because Ubuntu is based on debian.

Most specificity is preferred, so if both `deps_debian` and `deps_ubuntu` is provided, Ubuntu and all Ubuntu-based distros will use `deps_ubuntu` while Debian and all Debian-based distros 
that are not Ubuntu-based will use `deps_debian`.

Like distros are disabled when using the `LURE_DISTRO` environment variable.

## Variables

Any variables marked with `(*)` are required

### name (*)

The `name` variable contains the name of the package described by the script.

### version (*)

The `version` variable contains the version of the package. This should be the same as the version used by the author upstream.

Versions are compared using the [rpmvercmp](https://fedoraproject.org/wiki/Archive:Tools/RPM/VersionComparison) algorithm.

### release (*)

The `release` number is meant to differentiate between different builds of the same package version, such as if the script is changed but the version stays the same. The `release` must be an integer.

### epoch

The `epoch` number forces the package to be considered newer than versions with a lower epoch. It is meant to be used if the versioning scheme can't be used to determine which package is newer. Its use is discouraged and it should only be used if necessary. The `epoch` must be a positive integer.

### desc

The `desc` field contains the description for the package. It should not contain any newlines.

### homepage

The `homepage` field contains the URL to the website of the project packaged by this script.

### maintainer

The `maintainer` field contains the name and email address of the person maintaining the package. Example:

```text
Arsen Musayelyan <arsen@arsenm.dev>
```

While LURE does not require this field to be set, Debian has deprecated unset maintainer fields, and may disallow their use in `.deb` packages in the future.

### architectures

The `architectures` array contains all the architectures that this package supports. These match Go's GOARCH list, except for a few differences.

The `all` architecture will be translated to the proper term for the packaging format. For example, it will be changed to `noarch` if building a `.rpm`, or `any` if building an Arch package.

Since multiple variations of the `arm` architecture exist, the following values should be used:

`arm5`: armv5
`arm6`: armv6
`arm7`: armv7

LURE will attempt to detect which variant your system is using by checking for the existence of various CPU features. If this yields the wrong result or if you simply want to build for a different variant, the `LURE_ARM_VARIANT` variable should be set to the ARM variant you want. Example:

```shell
LURE_ARM_VARIANT=arm5 lure install ...
```

### licenses

The `licenses` array contains the licenses used by this package. In order to standardize license names, values should be [SPDX Identifiers](https://spdx.org/licenses/) such as `Apache-2.0`, `MIT`, and `GPL-3.0-only`. If the project uses a license that is not standardized in SPDX, use the value `Custom`. If the project has multiple nonstandard licenses, include `Custom` as many times as there are nonstandard licenses.

### provides

The `provides` array specifies what features the package provides. For example, if two packages build `ffmpeg` with different build flags, they should both have `ffmpeg` in the `provides` array. 

### conflicts

The `conflicts` array contains names of packages that conflict with the one built by this script. If two different packages contain the executable for `ffmpeg`, they cannot be installed at the same time, so they conflict. The `provides` array will also be checked, so this array generally contains the same values as `provides`.

### deps

The `deps` array contains the dependencies for the package. LURE repos will be checked first, and if the packages exist there, they will be built and installed. Otherwise, they will be installed from the system repos by your package manager.

### build_deps

The `build_deps` array contains the dependencies that are required to build the package. They will be installed before the build starts. Similarly to the `deps` array, LURE repos will be checked first.

### replaces

The `replaces` array contains the packages that are replaced by this package. Generally, if package managers find a package with a `replaces` field set, they will remove the listed package(s) and install that one instead. This is only useful if the packages are being stored in a repo for your package manager.

### sources

The `sources` array contains URLs which are downloaded into `$srcdir` before the build starts.

If the URL provided is an archive or compressed file, it will be extracted. To disable this, add the `~archive=false` query parameter. Example:

Extracted:
```text
https://example.com/archive.tar.gz
```

Not extracted:
```text
https://example.com/archive.tar.gz?~archive=false
```

If the URL scheme starts with `git+`, the source will be downloaded as a git repo. The git download mode supports multiple parameters:

- `~tag`: Specify which tag of the repo to check out.
- `~branch`: Specify which branch of the repo to check out.
- `~commit`: Specify which commit of the repo to check out.
- `~depth`: Specify what depth should be used when cloning the repo. Must be an integer.
- `~name`: Specify the name of the directory into which the git repo should be cloned.

Examples:

```text
git+https://gitea.arsenm.dev/Arsen6331/itd?~branch=resource-loading&~depth=1
```

```text
git+https://gitea.arsenm.dev/Arsen6331/lure?~tag=v0.0.1
```

### checksums

The `checksums` array must be the same length as the `sources` array. It contains sha256 checksums for the source files. The files are checked against the checksums and the build fails if they don't match.

To skip the check for a particular source, set the corresponding checksum to `SKIP`.

### backup

The `backup` array contains files that should be backed up when upgrading and removing. The exact behavior of this depends on your package manager. All files within this array must be full destination paths. For example, if there's a config called `config` in `/etc` that you want to back up, you'd set it like so:

```bash
backup=('/etc/config')
```

### scripts

The `scripts` variable contains a Bash associative array that specifies the location of various scripts relative to the build script. Example:

```bash
scripts=(
    ['preinstall']='preinstall.sh'
    ['postinstall']='postinstall.sh'
    ['preremove']='preremove.sh'
    ['postremove']='postremove.sh'
    ['preupgrade']='preupgrade.sh'
    ['postupgrade']='postupgrade.sh'
    ['pretrans']='pretrans.sh'
    ['posttrans']='posttrans.sh'
)
```

Note: The quotes are required due to limitations with the bash parser used.

The `preupgrade` and `postupgrade` scripts are only available in `.apk` and Arch Linux packages.

The `pretrans` and `posttrans` scripts are only available in `.rpm` packages.

The rest of the scripts are available in all packages.

---

## Functions

This section documents user-defined functions that can be added to build scripts. Any functions marked with `(*)` are required.

All functions are executed in the `$srcdir` directory

### version

The `version()` function updates the `version` variable. This allows for automatically deriving the version from sources. This is most useful for git packages, which usually don't need to be changed, so their `version` variable stays the same.

An example of using this for git:

```bash
version() {
	cd "$srcdir/itd"
	printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}
```

The AUR equivalent is the [`pkgver()` function](https://wiki.archlinux.org/title/VCS_package_guidelines#The_pkgver()_function)

### prepare

The `prepare()` function is meant to prepare the sources for building and packaging. This is the function in which patches should be applied, for example, by the `patch` command, and where tools like `go generate` should be executed.

### build

The `build()` function is where the package is actually built. Use the same commands that would be used to manually compile the software. Often, this function is just one line:

```bash
build() {
    make
}
```

### package (*)

The `package()` function is where the built files are placed into the directory that will be used by LURE to build the package.

Any files that should be installed on the filesystem should go in the `$pkgdir` directory in this function. For example, if you have a binary called `bin` that should be placed in `/usr/bin` and a config file called `bin.cfg` that should be placed in `/etc`, the `package()` function might look like this:

```bash
package() {
    install -Dm755 bin ${pkgdir}/usr/bin/bin
    install -Dm644 bin.cfg ${pkgdir}/etc/bin.cfg
}
```

---

## Environment Variables
 
LURE exposes several values as environment variables for use in build scripts.

### DISTRO_NAME

The `DISTRO_NAME` variable is the name of the distro as defined in its `os-release` file.

For example, it's set to `Fedora Linux` in a Fedora 36 docker image

### DISTRO_PRETTY_NAME

The `DISTRO_PRETTY_NAME` variable is the "pretty" name of the distro as defined in its `os-release` file.

For example, it's set to `Fedora Linux 36 (Container Image)` in a Fedora 36 docker image

### DISTRO_ID

The `DISTRO_ID` variable is the identifier of the distro as defined in its `os-release` file. This is the same as what LURE uses for overrides.

For example, it's set to `fedora` in a Fedora 36 docker image

### DISTRO_ID_LIKE

The `DISTRO_ID_LIKE` variable contains identifiers of similar distros to the one running, separated by spaces.

For example, it's set to `opensuse suse` in an OpenSUSE Tumbleweed docker image and `rhel fedora` in a CentOS 8 docker image.

### DISTRO_VERSION_ID

The `DISTRO_VERSION_ID` variable is the version identifier of the distro as defined in its `os-release` file.

For example, it's set to `36` in a Fedora 36 docker image and `11` in a Debian Bullseye docker image

### ARCH

The `ARCH` variable is the architecture of the machine running the script. It uses the same naming convention as the values in the `architectures` array

### NCPU

The `NCPU` variable is the amount of CPUs available on the machine running the script. It will be set to `8` on a quad core machine with hyperthreading, for example.

---

## Helper Commands

LURE provides various commands to help packagers create proper cross-distro packages. These commands should be used wherever possible instead of doing the tasks manually.

### install-binary

`install-binary` accepts 1-2 arguments. The first argument is the binary you'd like to install. The second is the filename that should be used.

If the filename argument is not provided, tha name of the input file will be used.

Examples:

```bash
install-binary ./itd
install-binary ./itd itd-2
```

### install-systemd

`install-systemd` installs regular systemd system services (see `install-systemd-user` for user services)

It accepts 1-2 arguments. The first argument is the service you'd like to install. The second is the filename that should be used.

If the filename argument is not provided, tha name of the input file will be used.

Examples:

```bash
install-systemd ./syncthing@.service
install-systemd-user ./syncthing@.service sync-thing@.service
```

### install-systemd-user

`install-systemd-user` installs systemd user services (services like `itd` meant to be started with `--user`).

It accepts 1-2 arguments. The first argument is the service you'd like to install. The second is the filename that should be used.

If the filename argument is not provided, tha name of the input file will be used.

Examples:

```bash
install-systemd-user ./itd.service
install-systemd-user ./itd.service infinitime-daemon.service
```

### install-config

`install-config` installs configuration files into the `/etc` directory

It accepts 1-2 arguments. The first argument is the config you'd like to install. The second is the filename that should be used.

If the filename argument is not provided, tha name of the input file will be used.

Examples:

```bash
install-config ./itd.toml
install-config ./itd.example.toml itd.toml
```

### install-license

`install-license` installs a license file

It accepts 1-2 arguments. The first argument is the config you'd like to install. The second is the filename that should be used.

If the filename argument is not provided, tha name of the input file will be used.

Examples:

```bash
install-license ./LICENSE itd/LICENSE
```

### install-completion

`install-completion` installs shell completions

It currently supports `bash`, `zsh`, and `fish`

Completions are read from stdin, so they can either be piped in or retrieved from files

Two arguments are required for this function. The first one is the name of the shell and the second is the name of the completion.

Examples:

```bash
./k9s completion fish | install-completion fish k9s
install-completion bash k9s <./k9s/completions/k9s.bash
```

### install-manual

`install-manual` installs manpages. It accepts a single argument, which is the path to the manpage.

The install path will be determined based on the number at the end of the filename. If a number cannot be extracted, an error will be returned.

Examples:

```bash
install-manual ./man/strelaysrv.1
install-manual ./mdoc.7
```

### install-desktop

`install-desktop` installs desktop files for applications. It accepts 1-2 arguments. The first argument is the config you'd like to install. The second is the filename that should be used.

If the filename argument is not provided, tha name of the input file will be used.

Examples:

```bash
install-desktop ./${name}/share/admc.desktop
install-desktop ./${name}/share/admc.desktop admc-app.desktop
```

### install-library

`install-library` installs shared and static libraries to the correct location.

This is the most important helper as it contains logic to figure out where to install libraries based on the target distro and CPU architecture. It should almost always be used to install all libraries.

It accepts 1-2 arguments. The first argument is the config you'd like to install. The second is the filename that should be used.

If the filename argument is not provided, tha name of the input file will be used.

Examples:

```bash
install-library ./${name}/build/libadldap.so
```

### git-version

`git-version` returns a version number based on the git revision of a repository.

If an argument is provided, it will be used as the path to the repo. Otherwise, the current directory will be used.

The version number will be the amount of revisions, a dot, and the short hash of the current revision. For example: `118.e4b8348`.

The AUR's convention includes an `r` at the beginning of the version number. This is ommitted because some distros expect the version number to start with a digit.

Examples:

```bash
git-version
git-version "$srcdir/itd"
```