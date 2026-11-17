// Package ports declares what the use cases of the plugin need from the machine.
package ports

// TempDirs creates the folder the runs start in and removes it.
type TempDirs interface {
	Create() (string, error)
	Remove(dir string) error
}
