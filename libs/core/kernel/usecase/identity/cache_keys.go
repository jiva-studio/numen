package identity

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// CacheKeys returns the key store that reads the identity record from store once and keeps it until the record is
// written or deleted through the returned store. Other keys and an absent record are read from store each time, so
// every writer of the record must write through the returned store.
func CacheKeys(store pluginport.KeyStore) pluginport.KeyStore {
	if cached, ok := store.(*cachedKeys); ok {
		return cached
	}
	return &cachedKeys{KeyStore: store}
}

type cachedKeys struct {
	pluginport.KeyStore

	mu     sync.Mutex
	record string
	isHeld bool
}

func (k *cachedKeys) GetKey(ctx context.Context, name string) (string, error) {
	if name != keyName {
		return k.KeyStore.GetKey(ctx, name)
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.isHeld {
		return k.record, nil
	}
	value, err := k.KeyStore.GetKey(ctx, name)
	if err != nil {
		return "", err
	}
	k.record, k.isHeld = value, true
	return value, nil
}

func (k *cachedKeys) SetKey(ctx context.Context, name, key string) error {
	if name != keyName {
		return k.KeyStore.SetKey(ctx, name, key)
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	k.record, k.isHeld = "", false
	return k.KeyStore.SetKey(ctx, name, key)
}

func (k *cachedKeys) DeleteKey(ctx context.Context, name string) error {
	if name != keyName {
		return k.KeyStore.DeleteKey(ctx, name)
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	k.record, k.isHeld = "", false
	return k.KeyStore.DeleteKey(ctx, name)
}
