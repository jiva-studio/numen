package filesystem

import (
	"errors"
	"io/fs"
	"time"
)

// Identity gives folders the identity that makes them vaults, and answers
// whether a folder can be read as one at all.
type Identity struct{ Options Options }

func (i Identity) Readable(root string) error {
	_, err := Open(root, i.Options)
	return err
}

func (i Identity) Ensure(root string, at time.Time) (string, error) {
	cfg, err := Initialize(root, i.Options.ServiceDir, at)
	if err != nil {
		return "", err
	}
	return cfg.ID, nil
}

// Of reads the identity a folder carries without creating one. A folder that is
// gone, or was never a vault, simply carries none — that is an answer rather
// than a failure.
func (i Identity) Of(root string) (string, bool, error) {
	cfg, err := ReadConfig(root, i.Options.ServiceDir)
	if errors.Is(err, ErrNotAVault) {
		return "", false, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return cfg.ID, true, nil
}
