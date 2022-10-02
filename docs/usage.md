# Usage

## Table of Contents

- [Commands](#commands)
    - [install](#install)
    - [remove](#remove)
    - [upgrade](#upgrade)
    - [info](#info)
    - [list](#list)
    - [build](#build)
    - [addrepo](#addrepo)
    - [removerepo](#removerepo)
    - [refresh](#refresh)
- [Environment Variables](#environment-variables)
    - [LURE_DISTRO](#lure_distro)
    - [LURE_PKG_FORMAT](#lure_pkg_format)
    - [LURE_ARM_VARIANT](#lure_arm_variant)

---

## Commands

### install

The install command installs a command from the LURE repos. Any packages that aren't found in LURE's repos get forwarded to the system package manager for installation.

Example:

```shell
lure in itd-bin
```

### remove

The remove command is for convenience. All it does is forwards the remove command to the system package manager.

Example:

```shell
lure rm firefox
```

### upgrade

The upgrade command looks through the packages installed on your system and sees if any of them match LURE repo packages. If they do, their versions are compared using the `rpmvercmp` algorithm. If LURE repos contain a newer version, the package is upgraded.

Example:

```shell
lure up
```

### info

The info command displays information about a package in LURE's repos.

Example:

```shell
lure info itd-bin
```

### list

The list command lists all LURE repo packages as well as their versions

Example:

```shell
lure ls
```

### build

The build command builds a package using a `lure.sh` build script in the current directory. The path to the script can be changed with the `-s` flag.

Example:

```shell
lure build
```

### addrepo

The addrepo command adds a repository to LURE if it doesn't already exist. The `-n` flag sets the name of the repository, and the `-u` flag is the URL to the repository. Both are required.

Example:

```shell
lure ar -n default -u https://github.com/Arsen6331/lure-repo
```

### removerepo

The removerepo command removes a repository from LURE and deletes its contents if it exists. The `-n` flag specifies the name of the repo to be deleted.

Example:

```shell
lure rr -n default
```

### refresh

The refresh command pulls all changes from all LURE repos that have changed.

Example:

```shell
lure ref
```

---

## Environment Variables

### LURE_DISTRO

The `LURE_DISTRO` environment variable should be set to the distro for which the package should be built. It tells LURE which overrides to use. Values should be the same as the `ID` field in `/etc/os-release` or `/usr/lib/os-release`. Possible values include:

- `arch`
- `alpine`
- `opensuse`
- `debian`

### LURE_PKG_FORMAT

The `LURE_PKG_FORMAT` environment variable should be set to the packaging format that should be used. Valid values are:

- `archlinux`
- `apk`
- `rpm`
- `deb`

### LURE_ARM_VARIANT

The `LURE_ARM_VARIANT` environment variable dictates which ARM variant to build for, if LURE is running on an ARM system. Possible values include:

- `arm5`
- `arm6`
- `arm7`

---

## Cross-packaging for other Distributions

You can create packages for different distributions  
setting the environment variables `LURE_DISTRO` and `LURE_PKG_FORMAT` as mentioned above.

Examples:

```
LURE_DISTRO=arch     LURE_PKG_FORMAT=archlinux lure build
LURE_DISTRO=alpine   LURE_PKG_FORMAT=apk       lure build
LURE_DISTRO=opensuse LURE_PKG_FORMAT=rpm       lure build
LURE_DISTRO=debian   LURE_PKG_FORMAT=deb       lure build
```

---