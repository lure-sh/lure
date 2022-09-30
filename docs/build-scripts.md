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

The `licenses` array contains the licenses used by this package. Some valid values include `GPLv3` and `MIT`.

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