package settings

import "errors"

// lockSuffix is added to the name of a settings file to name the file its writers lock. The settings
// file itself cannot be locked, since every write renames a new file over it.
const lockSuffix = ".lock"

func withLock(path string, work func() error) (err error) {
	release, err := lockFile(path + lockSuffix)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	return work()
}
