package ports

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// HostVault is the calls the plugin makes on one vault of the host, each through the narrow
// interface the adapter that uses it needs.
type HostVault struct {
	Files    host.FileReader
	Writer   host.FileWriter
	Index    host.VaultIndex
	Notes    host.NoteLookup
	Links    host.LinkLookup
	Cache    host.CacheStore
	Selector host.VaultSelector
}

// HostFolder is the plugin's folder in the store of a vault, as the review logs use it.
type HostFolder interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Append(ctx context.Context, name string, content []byte) error
	List(ctx context.Context) ([]host.StoreFile, error)
}
