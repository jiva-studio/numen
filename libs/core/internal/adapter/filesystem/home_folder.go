package filesystem

import "os"

// HomeFolder is the home directory the operating system gives the person.
type HomeFolder struct{}

func (HomeFolder) GetHomeFolder() (string, error) { return os.UserHomeDir() }
