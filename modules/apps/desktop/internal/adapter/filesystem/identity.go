package filesystem

import "time"

// Identity gives folders the identity that makes them vaults, and answers
// whether a folder can be read as one at all.
type Identity struct{ ServiceDir string }

func (i Identity) Readable(root string) error {
	_, err := Open(root, i.ServiceDir)
	return err
}

func (i Identity) Ensure(root string, at time.Time) (string, error) {
	cfg, err := Initialize(root, i.ServiceDir, at)
	if err != nil {
		return "", err
	}
	return cfg.ID, nil
}
