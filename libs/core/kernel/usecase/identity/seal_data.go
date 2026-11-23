package identity

import (
	"context"
	"io"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// SealData encrypts data of a plugin with the account keys of this device.
type SealData struct {
	records records
	random  io.Reader
}

// NewSealData returns the use case that seals data for a plugin.
func NewSealData(d Deps) *SealData { return &SealData{records: d.records, random: d.random} }

// Execute seals plain for the plugin and the context with the keys of the record, which it unwraps for the
// call only. It returns identity.ErrNoIdentity when the key store holds no record.
func (u *SealData) Execute(ctx context.Context, pluginID string, dataContext, plain []byte) ([]byte, error) {
	keys, err := readAccountKeys(ctx, u.records, pluginID)
	if err != nil {
		return nil, err
	}
	return keys.SealForPlugin(u.random, pluginID, dataContext, plain)
}

func readAccountKeys(ctx context.Context, records records, pluginID string) (identitylib.AccountKeys, error) {
	if pluginID == "" {
		return identitylib.AccountKeys{}, errNoPluginID
	}
	state, err := records.readState(ctx)
	if err != nil {
		return identitylib.AccountKeys{}, err
	}
	return state.OpenAccountKeys()
}
