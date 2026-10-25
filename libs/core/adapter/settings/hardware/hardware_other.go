//go:build !linux && !darwin && !windows

package hardware

// readPlatformHardware reports nothing; every field it leaves at zero is unknown.
func readPlatformHardware() Specs { return Specs{} }
