package identity

import (
	"context"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// SealData encrypts data of a plugin with the account keys of this device.
type SealData struct{ deps Deps }

// NewSealData returns the use case that seals data for a plugin.
func NewSealData(d Deps) *SealData { return &SealData{deps: d} }

// Execute seals plain for the plugin and the context with the keys of the record, which it unwraps for the
// call only. It returns identity.ErrNoIdentity when the key store holds no record.
func (u *SealData) Execute(ctx context.Context, pluginID string, dataContext, plain []byte) ([]byte, error) {
	keys, err := readAccountKeys(ctx, u.deps, pluginID)
	if err != nil {
		return nil, err
	}
	return keys.SealForPlugin(u.deps.random, pluginID, dataContext, plain)
}

func readAccountKeys(ctx context.Context, deps Deps, pluginID string) (identitylib.AccountKeys, error) {
	if pluginID == "" {
		return identitylib.AccountKeys{}, errNoPluginID
	}
	record, err := readExistingRecord(ctx, deps)
	if err != nil {
		return identitylib.AccountKeys{}, err
	}
	return record.Device.UnwrapAccountKeys(record.Wrapped)
}
