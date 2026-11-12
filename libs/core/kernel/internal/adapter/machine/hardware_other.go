//go:build !linux && !darwin && !windows

package machine

import machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"

func readPlatformHardware() machinedomain.Specs { return machinedomain.Specs{} }
