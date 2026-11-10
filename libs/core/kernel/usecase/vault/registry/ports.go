package registry

import "github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"

// Ports is the list of vaults an installation keeps, as the ports the use cases
// take. Each field is one operation of the list, and all of them answer from
// the same list.
type Ports struct {
	Vaults   vaultport.VaultList
	Lookup   vaultport.VaultLookup
	Saver    vaultport.VaultSaver
	Remover  vaultport.VaultRemover
	Recorder vaultport.VaultRecorder
	Last     vaultport.LastVault
}
