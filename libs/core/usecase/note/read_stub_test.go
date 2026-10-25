package note_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// statedReader is a vault that answers a stat and a read with whatever it was
// told to.
type statedReader struct {
	vaultport.VaultReader
	ref     domain.Fingerprint
	statErr error
	raw     []byte
	readErr error
}

func (r statedReader) GetFingerprint(context.Context, string) (domain.Fingerprint, error) {
	return r.ref, r.statErr
}

func (r statedReader) Read(context.Context, string) ([]byte, error) {
	return r.raw, r.readErr
}

type statedReaders struct{ reader statedReader }

func (r statedReaders) Open(domain.Vault) (vaultport.VaultReader, error) { return r.reader, nil }

func readStated(t *testing.T, reader statedReader) (note.Contents, error) {
	t.Helper()
	return note.NewRead(statedReaders{reader: reader}).Execute(t.Context(), domain.Vault{}, "Entropy.md")
}

const ceiling = 1048576

func TestTheCeilingIsOneMebibyteInclusive(t *testing.T) {
	t.Parallel()
	body := []byte("# Entropy\n")

	atCeiling, err := readStated(t, statedReader{
		ref: domain.Fingerprint{Path: "Entropy.md", Kind: domain.KindNote, Size: ceiling},
		raw: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if atCeiling.Outcome != note.Ok {
		t.Errorf("a note of exactly the ceiling is %q", atCeiling.Outcome)
	}

	finished, err := readStated(t, statedReader{
		ref: domain.Fingerprint{Path: "Entropy.md", Kind: domain.KindNote, Size: ceiling + 1},
		raw: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if finished.Outcome != note.TooLarge {
		t.Errorf("a note one byte over the ceiling is %q", finished.Outcome)
	}
	if finished.Body != "" {
		t.Errorf("a note over the ceiling came back with prose: %q", finished.Body)
	}
}

func TestAVaultThatCannotSayWhatIsAtAPathIsAnError(t *testing.T) {
	t.Parallel()
	broken := errors.New("the disk is gone")

	got, err := readStated(t, statedReader{statErr: broken, raw: []byte("# Entropy\n")})
	if !errors.Is(err, broken) {
		t.Fatalf("want the vault's own error, got %v", err)
	}
	if got != (note.Contents{}) {
		t.Errorf("an error came back with contents: %+v", got)
	}
}

// A file that appears between the question and the read is answered by its
func TestAFileThatAppearsAfterItWasAskedAboutIsNotANote(t *testing.T) {
	t.Parallel()

	got, err := readStated(t, statedReader{statErr: fs.ErrNotExist, raw: []byte("# Entropy\n")})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != note.NotANote {
		t.Errorf("want not a note, got %q", got.Outcome)
	}
	if got.Body != "" {
		t.Errorf("a file the vault does not hold as a note came back with prose: %q", got.Body)
	}
}

func TestAReadThatFailsAfterAMissingStatIsAnError(t *testing.T) {
	t.Parallel()
	broken := errors.New("the disk is gone")

	_, err := readStated(t, statedReader{statErr: fs.ErrNotExist, readErr: broken})
	if !errors.Is(err, broken) {
		t.Fatalf("want the vault's own error, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "Entropy.md") {
		t.Errorf("the error does not say which note: %v", err)
	}
}

type unopenableReaders struct{ err error }

func (r unopenableReaders) Open(domain.Vault) (vaultport.VaultReader, error) { return nil, r.err }

func TestAVaultThatCannotBeOpenedIsAnError(t *testing.T) {
	t.Parallel()
	broken := errors.New("the vault is gone")

	got, err := note.NewRead(unopenableReaders{err: broken}).Execute(t.Context(), domain.Vault{}, "Entropy.md")
	if !errors.Is(err, broken) {
		t.Fatalf("want the vault's own error, got %v", err)
	}
	if got != (note.Contents{}) {
		t.Errorf("an error came back with contents: %+v", got)
	}
}

func TestAPathTheVaultRefusesAsANoteIsNotANote(t *testing.T) {
	t.Parallel()

	got, err := readStated(t, statedReader{statErr: vaultport.ErrNotANote, raw: []byte("# Entropy\n")})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome != note.NotANote || got.Path != "Entropy.md" || got.Body != "" {
		t.Errorf("want not a note at Entropy.md with no prose, got %+v", got)
	}
}
