# Package Conventions

## General

Packages should have the name(s) of what they contain in their `provides` and `conflicts` arrays. That way, they can be installed by users without needing to know the full package name. For example, there are two LURE packages for ITD: `itd-bin`, and `itd-git`. Both of them have provides and conflicts arrays specifying the two commands they install: `itd`, and `itctl`. This means that if a user wants to install ITD, they simply have to type `lure in itd` and LURE will prompt them for which one they want to install.

## Binary packages

Packages that install download and install precompiled binaries should have a `-bin` suffix.

## Git packages

Packages that build and install programs from source code cloned directly from Git should have a `-git` suffix.

The versions of these packages should consist of the amount of revisions followed by the current revision, separated by a period. For example: `183.80187b0`. Note that unlike the AUR, there is no `r` at the beginning. This is because some package managers refuse to install packages whose version numbers don't start with a digit.

This version number can be obtained using the following command:

```bash
printf "%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
```

The `version()` function for such packages should use the LURE-provided `git-version` helper command, like so:

```bash
version() {
    cd "$srcdir/$name"
    git-version
}
```

This uses LURE's embedded Git implementation, which ensures that the user doesn't need Git installed on their system in order to install `-git` packages.

## Other packages

Packages that download sources for a specific version of a program should not have any suffix, even if those sources are downloaded from Git.