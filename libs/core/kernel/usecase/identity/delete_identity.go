package identity

import "context"

type DeleteIdentity struct{ deps Deps }

// NewDeleteIdentity returns the use case that removes the identity of this device.
func NewDeleteIdentity(d Deps) *DeleteIdentity { return &DeleteIdentity{deps: d} }

// Execute removes the record; a missing record is not an error.
func (u *DeleteIdentity) Execute(ctx context.Context) error {
	if err := u.deps.keys.DeleteKey(ctx, keyName); err != nil && !isNoKey(err) {
		return err
	}
	return nil
}
