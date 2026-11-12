// Package vaultaddress finds the vault an agent names by its identifier, name or folder.
package vaultaddress

import (
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// Find returns the first entry whose identifier is the address, else the first whose name is, else the
// first whose folder is the same folder as the address.
func Find(entries []host.VaultEntry, address string) (host.VaultEntry, bool) {
	for _, isMatch := range []func(host.VaultEntry) bool{
		func(entry host.VaultEntry) bool { return string(entry.ID) == address },
		func(entry host.VaultEntry) bool { return entry.Name == address },
		func(entry host.VaultEntry) bool { return filepath.Clean(entry.Path) == filepath.Clean(address) },
	} {
		for _, entry := range entries {
			if isMatch(entry) {
				return entry, true
			}
		}
	}
	return host.VaultEntry{}, false
}
