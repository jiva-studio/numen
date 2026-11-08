//go:build !linux && !darwin && !windows

package hardware

func readPlatformHardware() Specs { return Specs{} }
