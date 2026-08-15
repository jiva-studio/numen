package filesystem

import "time"

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
