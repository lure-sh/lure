# Configuration

This page describes the configuration of LURE

---

## Table of Contents

- [Config file](#config-file)
    - [rootCmd](#rootcmd)
    - [repo](#repo)

---

## File locations

| Path | Description 
| --:  | :--
| ~/.config/lure/lure.toml | Config file
| ~/.cache/lure/pkgs       | here the packages are built and stored
| ~/.cache/lure/repo       | here are the git repos with all the `lure.sh` files  
|                          | Example: `~/.cache/lure/repo/default/itd-bin/lure.sh`

---

## Config file

### rootCmd

The `rootCmd` field in the config specifies which command should be used for privilege elevation. The default value is `sudo`.

### repo

The `repo` array in the config specifies which repos are added to LURE. Each repo must have a name and URL. A repo looks like this in the config:

```toml
[[repo]]
name = 'default'
url = 'https://github.com/Elara6331/lure-repo.git'
```

The `default` repo is added by default. Any amount of repos may be added.

---
