// Package settags toggles tags on notes: it reads the tags the notes have, computes the new lists and writes them to the
// frontmatter of the notes through the host.
package settags

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
)

// Reader returns the tags of every note of a vault that has tags, by path.
type Reader interface {
	GetMap(ctx context.Context, id domain.VaultID) (map[string][]string, error)
}

// PropertyWriter writes the frontmatter of notes.
type PropertyWriter interface {
	WriteFrontmatter(ctx context.Context, writes []host.FrontmatterWrite) ([]host.FrontmatterResult, error)
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

// recentFile is the file of the plugin's store folder the recent tags are kept in, and recentLimit the most it holds.
const (
	recentFile  = "recent.json"
	recentLimit = 20
)

// change is the toggles one note receives and the write that makes them.
type change struct {
	index   int
	toggles []toggle
	write   host.FrontmatterWrite
}

// Execute toggles each tag of toggles on the notes at paths and returns one Result for each path, in order. With
// several notes a tag that every note has is removed from all, and any other is set on all. A note left with no tag
// loses the key tags. A note that cannot be written costs the others nothing. It remembers the tags it set as recent.
func (u *UseCase) Execute(ctx context.Context, id domain.VaultID, paths, toggles []string) ([]Result, error) {
	current, err := u.reader.GetMap(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the tags of the notes: %w", err)
	}
	decided := decideToggles(paths, current, toggles)
	results := make([]Result, len(paths))
	var changes []change
	for i, path := range paths {
		received := pickToggles(decided, current[path])
		results[i] = Result{Path: path, Tags: tags.Apply(current[path], getTags(received)), Err: nil}
		if len(received) > 0 {
			values := map[string]any{tags.FrontmatterKey: getValue(results[i].Tags)}
			changes = append(changes, change{index: i, toggles: received, write: host.FrontmatterWrite{Path: path, Values: values}})
		}
	}
	vault := u.open(id)
	outcomes, err := writeChanges(ctx, vault.Properties, changes)
	if err != nil {
		return nil, err
	}
	var set []toggle
	for n, outcome := range outcomes {
		one := changes[n]
		if outcome.Err != nil {
			results[one.index].Tags, results[one.index].Err = slices.Clone(current[results[one.index].Path]), outcome.Err
			continue
		}
		for _, applied := range one.toggles {
			if !applied.isClear {
				set = append(set, applied)
			}
		}
	}
	return results, remember(ctx, vault.Store, set)
}

// ListRecent returns the tags Execute set, the latest first, each once.
func (u *UseCase) ListRecent(ctx context.Context, id domain.VaultID) ([]string, error) {
	return readRecent(ctx, u.open(id).Store)
}

func writeChanges(ctx context.Context, writer PropertyWriter, changes []change) ([]host.FrontmatterResult, error) {
	if len(changes) == 0 {
		return nil, nil
	}
	writes := make([]host.FrontmatterWrite, len(changes))
	for i, one := range changes {
		writes[i] = one.write
	}
	outcomes, err := writer.WriteFrontmatter(ctx, writes)
	if err != nil {
		return nil, fmt.Errorf("write the tags of %d notes: %w", len(writes), err)
	}
	if len(outcomes) != len(writes) {
		return nil, fmt.Errorf("write the tags of %d notes: the host answered for %d", len(writes), len(outcomes))
	}
	return outcomes, nil
}

func getValue(list []string) any {
	if len(list) == 0 {
		return nil
	}
	return list
}
