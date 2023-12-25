#!/bin/bash

# LURE - Linux User REpository
# Copyright (C) 2023 Elara Musayelyan
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

info() {
  echo $'\x1b[32m[INFO]\x1b[0m' $@
}

warn() {
  echo $'\x1b[31m[WARN]\x1b[0m' $@
}

error() {
  echo $'\x1b[31;1m[ERR]\x1b[0m' $@
  exit 1
}

installPkg() {
  rootCmd=""
  if command -v doas &>/dev/null; then
    rootCmd="doas"
  elif command -v sudo &>/dev/null; then
    rootCmd="sudo"
  else
    warn "No privilege elevation command (e.g. sudo, doas) detected"
  fi
  
  case $1 in
  pacman) $rootCmd pacman --noconfirm -U ${@:2} ;;
  apk) $rootCmd apk add --allow-untrusted ${@:2} ;;
  zypper) $rootCmd zypper --no-gpg-checks install ${@:2} ;;
  *) $rootCmd $1 install -y ${@:2} ;;
  esac
}

if ! command -v curl &>/dev/null; then
  error "This script requires the curl command. Please install it and run again."
fi

pkgFormat=""
pkgMgr=""
if command -v pacman &>/dev/null; then
  info "Detected pacman"
  pkgFormat="pkg.tar.zst"
  pkgMgr="pacman"
elif command -v apt &>/dev/null; then
  info "Detected apt"
  pkgFormat="deb"
  pkgMgr="apt"
elif command -v dnf &>/dev/null; then
  info "Detected dnf"
  pkgFormat="rpm"
  pkgMgr="dnf"
elif command -v yum &>/dev/null; then
  info "Detected yum"
  pkgFormat="rpm"
  pkgMgr="yum"
elif command -v zypper &>/dev/null; then
  info "Detected zypper"
  pkgFormat="rpm"
  pkgMgr="zypper"
elif command -v apk &>/dev/null; then
  info "Detected apk"
  pkgFormat="apk"
  pkgMgr="apk"
else
  error "No supported package manager detected!"
fi

latestVersion=$(curl -sI 'https://gitea.elara.ws/lure/lure/releases/latest' | grep -io 'location: .*' | rev | cut -d '/' -f1 | rev | tr -d '[:space:]')
info "Found latest LURE version:" $latestVersion

fname="$(mktemp -u -p /tmp "lure.XXXXXXXXXX").${pkgFormat}"
url="https://gitea.elara.ws/lure/lure/releases/download/${latestVersion}/linux-user-repository-${latestVersion#v}-linux-$(uname -m).${pkgFormat}"

info "Downloading LURE package" 
curl -L $url -o $fname

info "Installing LURE package"
installPkg $pkgMgr $fname

info "Cleaning up"
rm $fname

info "Done!"
