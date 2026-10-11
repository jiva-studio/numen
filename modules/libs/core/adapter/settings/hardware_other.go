//go:build !linux && !darwin && !windows

package settings

// readPlatformHardware reports nothing; every field it leaves at zero is unknown.
func readPlatformHardware() HardwareSpecs { return HardwareSpecs{} }
