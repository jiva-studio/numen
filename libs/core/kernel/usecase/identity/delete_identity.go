package identity

import "context"

type DeleteIdentity struct{ records records }

// NewDeleteIdentity returns the use case that removes the identity of this device.
func NewDeleteIdentity(d Deps) *DeleteIdentity { return &DeleteIdentity{records: d.records} }

// Execute removes the record, then the chain file; a missing record is not an error.
func (u *DeleteIdentity) Execute(ctx context.Context) error {
	if err := u.records.keys.DeleteKey(ctx, keyName); err != nil && !isNoKey(err) {
		return err
	}
	return u.records.chains.Delete(ctx)
}
