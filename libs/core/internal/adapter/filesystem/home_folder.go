package filesystem

import "os"

// HomeFolder is the home directory the operating system reports for the user.
type HomeFolder struct{}

func (HomeFolder) GetHomeFolder() (string, error) { return os.UserHomeDir() }
