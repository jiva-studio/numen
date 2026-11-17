package transcribe

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/ports"
)

// Files reads files from the vaults known to the host.
type Files interface {
	ReadFile(ctx context.Context, vault domain.VaultID, path string) ([]byte, error)
}

// activeVaultProvider is implemented by Files that can return the vault the host shows. A recording
// requested with an empty vault ID is then claimed under the id of that vault, the id a tool call
// names.
type activeVaultProvider interface {
	GetActiveVaultID(ctx context.Context) (domain.VaultID, error)
}

// Folder is the plugin's folder in the store of a vault, as the transcripts use it.
type Folder interface {
	host.StoreWriter
	Read(ctx context.Context, name string) ([]byte, error)
}

// Vault is what a request checks of a vault: that the file named is there.
type Vault interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
}

// Engines opens the model that transcribes a recording. The model is opened once and reused.
type Engines func(ctx context.Context) (ports.Recognizer, error)
