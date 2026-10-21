package search

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// fakeVault is a vault held in memory, safe to read from several goroutines, whose Read counts every call it gets
// and the order they came in, so a test can see how many times a source was
// decoded and in what order.
type fakeVault struct {
	mu      sync.Mutex
	content map[string]string
	calls   map[string]int
	order   []string
}

func (f *fakeVault) WalkFiles(context.Context, func(domain.Fingerprint) error) error { return nil }

func (f *fakeVault) List(context.Context, string) ([]domain.Entry, error) { return nil, nil }

func (f *fakeVault) Open(context.Context, string) (io.ReadSeekCloser, error) {
	return nil, errors.New("not read a part at a time by this fixture")
}

func (f *fakeVault) GetFingerprint(_ context.Context, path string) (domain.Fingerprint, error) {
	if _, target := f.content[path]; !target {
		return domain.Fingerprint{}, fs.ErrNotExist
	}
	return domain.Fingerprint{Path: path, Kind: domain.KindNote}, nil
}

func (f *fakeVault) Read(_ context.Context, path string) ([]byte, error) {
	f.mu.Lock()
	f.calls[path]++
	f.order = append(f.order, path)
	f.mu.Unlock()
	raw, target := f.content[path]
	if !target {
		return nil, fs.ErrNotExist
	}
	return []byte(raw), nil
}

// fakeVaultReaders opens the one fakeVault given, for whichever vault is asked.
type fakeVaultReaders struct{ reader *fakeVault }

func (f fakeVaultReaders) Open(domain.Vault) (port.VaultReader, error) { return f.reader, nil }

// proseOf is a source's prose of the length given, holding the source's own
// name once per line so a line number pins the line a passage is read back on.
func proseOf(source string, lines int) string {
	var b strings.Builder
	for i := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(source)
	}
	return b.String()
}

func TestReadFillsEveryPassageOfASourceScatteredThroughFound(t *testing.T) {
	// A ranking scatters one source's passages wherever each chunk placed, so a
	// source that stood first can stand again after several others have been
	// read. Every passage still has to come back with the text and the line its
	// own offsets name, whichever source stood near it in `found`.
	ctx := t.Context()
	vault := &fakeVault{content: map[string]string{}, calls: map[string]int{}}
	sources := []string{"a.md", "b.md", "c.md", "d.md", "e.md"}
	for _, s := range sources {
		vault.content[s] = proseOf(s, 10)
	}

	u := Search{readers: fakeVaultReaders{reader: vault}}

	// "a.md" stands first and last; the four others stand between its two
	// passages, so it is read once and asked for again only from what that one
	// reading produced.
	found := []domain.Passage{
		{ChunkID: "1", Source: "a.md", Start: 0, Length: len("a.md")},
		{ChunkID: "2", Source: "b.md", Start: 0, Length: len("b.md")},
		{ChunkID: "3", Source: "c.md", Start: 0, Length: len("c.md")},
		{ChunkID: "4", Source: "d.md", Start: 0, Length: len("d.md")},
		{ChunkID: "5", Source: "e.md", Start: 0, Length: len("e.md")},
		// The second passage of "a.md" opens on its sixth line.
		{ChunkID: "6", Source: "a.md", Start: 5 * (len("a.md") + 1), Length: len("a.md")},
	}

	out, err := u.read(ctx, domain.Vault{ID: "v"}, found)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(found) {
		t.Fatalf("%d passages came back, want %d", len(out), len(found))
	}

	for i, p := range out {
		if p.ChunkID != found[i].ChunkID {
			t.Fatalf("passage %d is %s, want the output in found's own order: %+v", i, p.ChunkID, out)
		}
		if p.Text != p.Source {
			t.Errorf("%s came back with text %q, want its own name", p.ChunkID, p.Text)
		}
	}
	if out[0].Line != 0 {
		t.Errorf("the first passage of a.md stands on line %d, want 0", out[0].Line)
	}
	if out[5].Line != 5 {
		t.Errorf("the second passage of a.md stands on line %d, want 5", out[5].Line)
	}

	// The whole point of grouping by source: a.md's prose is decoded once, not
	// once per passage that names it, so nothing here ever holds more than one
	// source's decoded text at a time.
	for _, s := range sources {
		if vault.calls[s] != 1 {
			t.Errorf("%s was read %d times, want exactly 1", s, vault.calls[s])
		}
	}
	if vault.calls["a.md"] != 1 {
		t.Errorf("a.md was read %d times despite two scattered passages, want 1", vault.calls["a.md"])
	}

	// Every source's two reads would have to stand apart for one to be held
	// while another is read; they stand together, which is what one decode per
	// source, read once and used before the next begins, looks like.
	if got := vault.order; len(got) != len(sources) {
		t.Errorf("the vault was opened %d times for %d distinct sources: %v", len(got), len(sources), got)
	}
}

func TestReadDropsAPassageOfASourceTheVaultNoLongerHolds(t *testing.T) {
	ctx := t.Context()
	vault := &fakeVault{content: map[string]string{"here.md": "here.md"}, calls: map[string]int{}}
	u := Search{readers: fakeVaultReaders{reader: vault}}

	found := []domain.Passage{
		{ChunkID: "1", Source: "gone.md", Start: 0, Length: 4},
		{ChunkID: "2", Source: "here.md", Start: 0, Length: len("here.md")},
	}

	out, err := u.read(ctx, domain.Vault{ID: "v"}, found)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ChunkID != "2" {
		t.Fatalf("got %+v, want the one passage of the source still there", out)
	}
}
