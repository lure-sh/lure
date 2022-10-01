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
