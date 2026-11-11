package presets_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presets"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

func getVault() map[string]string {
	return map[string]string{
		"Term.md": "---\ntype: card-model\nfields:\n  - Word\n  - Meaning\n---\n" +
			"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n",
		"decks/Words.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mold ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mold\n" +
			"\n### Meaning\n\nCompost made of fallen leaves alone\n",
		"decks/Roots.md": "---\ntype: deck\n---\n" +
			"\n## Taproot ^k7m2xq9fzp\n\n[[Term]]\n\n### Word\n\nTaproot\n" +
			"\n### Meaning\n\nThe main root, growing straight down\n",
	}
}

// vaulted is one vault on disk, and the presets built over it.
type vaulted struct {
	vault   domain.Vault
	env     vaulttest.Env
	presets presets.Presets
	notes   vaulttest.Queries
	links   ports.LinkQueries
	scan    func(ctx context.Context, v domain.Vault, paths []string) error
}

func openVault(tb testing.TB, notes map[string]string) vaulted {
	tb.Helper()
	env := vaulttest.New(tb, notes)
	scanned := func(_ context.Context, _ domain.Vault, _ []string) error {
		env.Host.Reindex()
		return nil
	}

	deckPresets := presets.New(
		env.Ports.Readers, env.Ports.Writers,
		env.Queries, env.Queries, scanned, vaulttest.NewClock())
	deckPresets.Problems = env.Queries

	return vaulted{
		vault:   env.Vault,
		env:     env,
		presets: deckPresets,
		notes:   env.Queries,
		links:   env.Queries,
		scan:    scanned,
	}
}

func read(t *testing.T, v domain.Vault, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func busy(context.Context, domain.Vault, []string) error {
	return errors.New("the index is busy")
}

func getToday() review.Day {
	return review.Day{Starts: review.DayStartTime}
}
