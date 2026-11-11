package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
)

type pluginCache interface {
	Read(ctx context.Context, pluginID string, vaultID domain.VaultID, name string) ([]byte, bool, error)
	Write(ctx context.Context, pluginID string, vaultID domain.VaultID, name string, content []byte) error
}

// Stores opens the folder of a plugin in the store of the vault, and keeps the cache of a plugin for it.
type Stores struct {
	StoreReaders func(v domain.Vault, pluginID string) (vault.StoreReader, error)
	StoreWriters func(v domain.Vault, pluginID string) (vault.StoreWriter, error)
	Cache        pluginCache

	session sessionAccess
}

// OpenStore opens the folder of the plugin for reading.
func (s Stores) OpenStore(pluginID string) (vault.StoreReader, error) {
	v, err := s.session.open()
	if err != nil {
		return nil, err
	}
	return s.StoreReaders(v, pluginID)
}

// OpenStoreWriter opens the folder of the plugin for writing.
func (s Stores) OpenStoreWriter(pluginID string) (vault.StoreWriter, error) {
	v, err := s.session.open()
	if err != nil {
		return nil, err
	}
	return s.StoreWriters(v, pluginID)
}

// ReadCache returns what the plugin cached under the name for the vault.
func (s Stores) ReadCache(ctx context.Context, pluginID, name string) ([]byte, bool, error) {
	v, err := s.session.open()
	if err != nil {
		return nil, false, err
	}
	return s.Cache.Read(ctx, pluginID, v.ID, name)
}

// WriteCache keeps content for the plugin under the name for the vault.
func (s Stores) WriteCache(ctx context.Context, pluginID, name string, content []byte) error {
	v, err := s.session.open()
	if err != nil {
		return err
	}
	return s.Cache.Write(ctx, pluginID, v.ID, name, content)
}
