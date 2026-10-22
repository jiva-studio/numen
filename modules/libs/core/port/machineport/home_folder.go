package machineport

// HomeFolder is where the person's own files begin on this machine.
type HomeFolder interface {
	// GetHomeFolder returns the person's home directory.
	GetHomeFolder() (string, error)
}
