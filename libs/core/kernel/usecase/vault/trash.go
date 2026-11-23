package vault

// Trash is where a folder goes when the user is done with it.
type Trash interface {
	// MoveToTrash moves a folder to the location where this machine keeps deleted
	// items. It returns machine.ErrNoTrash where this machine has no such location.
	MoveToTrash(path string) error
}
