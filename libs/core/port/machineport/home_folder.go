package machineport

// HomeFolder is the location of the user's home directory on this machine.
type HomeFolder interface {
	// GetHomeFolder returns the user's home directory.
	GetHomeFolder() (string, error)
}
