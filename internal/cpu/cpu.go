/*
 * LURE - Linux User REpository
 * Copyright (C) 2022 Arsen Musayelyan
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
	"strings"

	"golang.org/x/sys/cpu"
)

// ARMVariant checks which variant of ARM lure is running
// on, by using the same detection method as Go itself
func ARMVariant() string {
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
