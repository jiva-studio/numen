package settings

import "errors"

// lockSuffix is added to the name of a settings file to name the file its writers lock. The settings
// file itself cannot be locked, since every write renames a new file over it.
const lockSuffix = ".lock"

// withLock runs work under an exclusive lock that every writer of the file takes, in this
// process and in any other, so that a write reads the file after the writes before it ended.
func withLock(path string, work func() error) (err error) {
	release, err := lockFile(path + lockSuffix)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	return work()
}
