/*
 * LURE - Linux User REpository
 * Copyright (C) 2023 Elara Musayelyan
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package cpu

import (
	"os"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/cpu"
)

// armVariant checks which variant of ARM lure is running
// on, by using the same detection method as Go itself
func armVariant() string {
	armEnv := os.Getenv("LURE_ARM_VARIANT")
	// ensure value has "arm" prefix, such as arm5 or arm6
	if strings.HasPrefix(armEnv, "arm") {
		return armEnv
	}

	if cpu.ARM.HasVFPv3 {
		return "arm7"
	} else if cpu.ARM.HasVFP {
		return "arm6"
	} else {
		return "arm5"
	}
}

// Arch returns the canonical CPU architecture of the system
func Arch() string {
	arch := os.Getenv("LURE_ARCH")
	if arch == "" {
		arch = runtime.GOARCH
	}
	if arch == "arm" {
		arch = armVariant()
	}
	return arch
}

func IsCompatibleWith(target string, list []string) bool {
	if target == "all" {
		return true
	}

	for _, arch := range list {
		if strings.HasPrefix(target, "arm") && strings.HasPrefix(arch, "arm") {
			targetVer, err := getARMVersion(target)
			if err != nil {
				return false
			}

			archVer, err := getARMVersion(arch)
			if err != nil {
				return false
			}

			if targetVer >= archVer {
				return true
			}
		}

		if target == arch {
			return true
		}
	}

	return false
}

func CompatibleArches(arch string) ([]string, error) {
	if strings.HasPrefix(arch, "arm") {
		ver, err := getARMVersion(arch)
		if err != nil {
			return nil, err
		}

		if ver > 5 {
			var out []string
			for i := ver; i >= 5; i-- {
				out = append(out, "arm"+strconv.Itoa(i))
			}
			return out, nil
		}
	}

	return []string{arch}, nil
}

func getARMVersion(arch string) (int, error) {
	// Extract the version number from ARM architecture
	version := strings.TrimPrefix(arch, "arm")
	if version == "" {
		return 5, nil // Default to arm5 if version is not specified
	}
	return strconv.Atoi(version)
}
