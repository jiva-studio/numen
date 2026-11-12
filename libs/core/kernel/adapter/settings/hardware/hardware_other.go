//go:build !linux && !darwin && !windows

package hardware

import settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

func readPlatformHardware() settingsdomain.MachineSpecs { return settingsdomain.MachineSpecs{} }
