package plugin

import (
	"sync"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// keyStoreHolder holds the key store the host reads values from, once the app has set it.
type keyStoreHolder struct {
	mu    sync.RWMutex
	store pluginport.KeyStore
}

func (k *keyStoreHolder) set(store pluginport.KeyStore) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.store = store
}

func (k *keyStoreHolder) get() pluginport.KeyStore {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.store
}

// SetKeyStore sets where the values of secrets are kept.
func (h *Host) SetKeyStore(store pluginport.KeyStore) {
	h.keyStore.set(store)
}
