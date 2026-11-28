package bind

import "github.com/jiva-studio/numen/apps/mobile/internal/keyvault"

// KeyVault is what the platform keeps the keys of the identity in: the key store of the operating system. A
// name with no key reads as the empty string and Has is false.
type KeyVault interface {
	Get(name string) (string, error)
	Has(name string) bool
	Set(name, value string) error
	Delete(name string) error
}

// SetKeyVault gives the core the vault of keys of the platform. It is called before Start; a core started
// without one has no identity, and with it no sync.
func (c *Core) SetKeyVault(vault KeyVault) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = vault
}

func (c *Core) getKeyStore() *keyvault.Store {
	if c.keys == nil {
		return nil
	}
	return keyvault.New(c.keys)
}
