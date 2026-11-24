package identity

// DeviceName gives the name a device is listed under in the device chain.
type DeviceName interface {
	// GetInstanceName returns the human-readable name of this machine.
	GetInstanceName() string
}
