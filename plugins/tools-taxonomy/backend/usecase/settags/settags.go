// Package settags toggles tags on notes: it reads the tags the notes have, computes the new lists and writes them to the
// frontmatter of the notes through the host.
package settags

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Reader returns the tags of every note of a vault that has tags, by path.
type Reader interface {
	GetMap(ctx context.Context, id domain.VaultID) (map[string][]string, error)
}

// PropertyWriter writes the frontmatter of notes.
type PropertyWriter interface {
	WriteFrontmatter(ctx context.Context, writes []host.FrontmatterWrite) ([]host.FrontmatterWritten, error)
}

// Store reads and writes a file of the plugin's folder of a vault's store.
type Store interface {
	Read(ctx context.Context, name string) ([]byte, error)
	Write(ctx context.Context, name string, content []byte) error
}

// Vault is what the use case writes to and keeps its list of recent tags in.
type Vault struct {
	Properties PropertyWriter
	Store      Store
}

// Result is the outcome for one note: the tags it has after the call, or the error that left it unchanged.
type Result struct {
	Path string
	Tags []string
	Err  error
}

// UseCase toggles tags on the notes of the vaults open returns.
type UseCase struct {
	reader Reader
	open   func(id domain.VaultID) Vault
}

// New returns the use case over the tags reader gives and the vaults open returns.
func New(reader Reader, open func(id domain.VaultID) Vault) *UseCase {
	return &UseCase{reader: reader, open: open}
}

// Execute toggles each tag of toggles on the notes at paths and returns one Result for each path, in order. With
// several notes a tag that every note has is removed from all, and any other is set on all. A note left with no tag
// loses the key tags. A note that cannot be written costs the others nothing. It remembers the tags it set as recent.
func (u *UseCase) Execute(ctx context.Context, id domain.VaultID, paths, toggles []string) ([]Result, error) {
	return nil, nil
}

// ListRecent returns the tags Execute set, the latest first, each once.
func (u *UseCase) ListRecent(ctx context.Context, id domain.VaultID) ([]string, error) {
	return nil, nil
}
