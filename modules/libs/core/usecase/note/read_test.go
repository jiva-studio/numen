package note_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// readable is a vault on disk and the read that works on it.
func readable(t *testing.T, notes map[string]string) (note.Read, domain.Vault) {
	t.Helper()
	return note.Read{Readers: filesystem.VaultReaders{}}, testsupport.NewVault(t, notes)
}

func read(t *testing.T, u note.Read, v domain.Vault, path string) note.Contents {
	t.Helper()
	contents, err := u.Execute(t.Context(), v, path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return contents
}

func TestAReadGivesTheProseAndWhatTheFileWas(t *testing.T) {
	t.Parallel()
	raw := "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n# Entropy\n\nA measure of disorder.\n"
	u, v := readable(t, map[string]string{"Entropy.md": raw})

	got := read(t, u, v, "Entropy.md")
	if got.Outcome != note.Ok {
		t.Fatalf("want ok, got %q", got.Outcome)
	}
	if got.Body != "# Entropy\n\nA measure of disorder.\n" {
		t.Errorf("the frontmatter came back as prose: %q", got.Body)
	}
	if got.Fingerprint.Size != int64(len(raw)) || got.Fingerprint.Path != "Entropy.md" {
		t.Errorf("the file was not described: %+v", got.Fingerprint)
	}
}

// A path with no file behind it is an answer. The tab that asked keeps what the
// person is reading, and the next write makes the note.
func TestAPathWithNoFileIsMissing(t *testing.T) {
	t.Parallel()
	u, v := readable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	got := read(t, u, v, "gone.md")
	if got.Outcome != note.Missing {
		t.Errorf("want missing, got %q", got.Outcome)
	}
	if got.Body != "" {
		t.Errorf("a missing note came back with prose: %q", got.Body)
	}
}

// The vault holds notes. Anything else in the folder — an export, an
// attachment, a book — is refused by name rather than handed over as prose.
func TestSomethingTheVaultDoesNotHoldAsANoteIsRefused(t *testing.T) {
	t.Parallel()
	u, v := readable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	book := "PK\x03\x04 chapters and chapters of somebody else's book"
	if err := os.WriteFile(filepath.Join(v.Path, "library.epub"), []byte(book), 0o644); err != nil {
		t.Fatal(err)
	}

	got := read(t, u, v, "library.epub")
	if got.Outcome != note.NotANote {
		t.Fatalf("want not a note, got %q", got.Outcome)
	}
	if got.Body != "" {
		t.Errorf("the bytes of a book were handed over: %q", got.Body)
	}
}

// The body goes out in a string field, and one byte that is not UTF-8 turns
// into U+FFFD wherever it is shown. The note is refused, and its bytes stay as
// the person left them.
func TestAFileThatIsNotTextIsRefused(t *testing.T) {
	t.Parallel()
	u, v := readable(t, map[string]string{"Entropy.md": "# Entropy\n\xff\xfe pasted\n"})

	got := read(t, u, v, "Entropy.md")
	if got.Outcome != note.NotText {
		t.Fatalf("want not text, got %q", got.Outcome)
	}
	if got.Body != "" {
		t.Errorf("bytes that are not text came back: %q", got.Body)
	}
}

func TestANoteOverTheCeilingIsRefused(t *testing.T) {
	t.Parallel()
	u, v := readable(t, map[string]string{
		"Entropy.md": "# Entropy\n" + strings.Repeat("a measure of disorder ", note.MaxBytes/20),
	})

	got := read(t, u, v, "Entropy.md")
	if got.Outcome != note.TooLarge {
		t.Fatalf("want too large, got %q", got.Outcome)
	}
	if got.Body != "" {
		t.Errorf("a note over the ceiling came back anyway: %d bytes", len(got.Body))
	}
	if got.Fingerprint.Size <= note.MaxBytes {
		t.Errorf("the size that was refused is not reported: %+v", got.Fingerprint)
	}
}

// A note whose frontmatter is not YAML can be neither read nor written from
// here, and the outcome says which note that is.
func TestANoteWhoseFrontmatterCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()
	u, v := readable(t, map[string]string{"Entropy.md": "---\nid: [unterminated\n---\n# Entropy\n"})

	got := read(t, u, v, "Entropy.md")
	if got.Outcome != note.Unreadable {
		t.Fatalf("want unreadable, got %q", got.Outcome)
	}
	if got.Body != "" {
		t.Errorf("a note that cannot be read came back with prose: %q", got.Body)
	}
}

// The prose a person is handed has one kind of line break in it. The file keeps
// its own: a chunk's offsets are byte offsets into it, so the bytes the index
// is built from are the bytes on disk.
func TestAReadNormalisesTheProseAndLeavesTheFileAlone(t *testing.T) {
	t.Parallel()
	raw := "---\r\nid: 01J8F3K2M9QRSTVWXYZ012\r\n---\r\n# Entropy\r\n\r\nA measure of disorder.\r\n"
	u, v := readable(t, map[string]string{"Entropy.md": raw})

	got := read(t, u, v, "Entropy.md")
	if strings.ContainsRune(got.Body, '\r') {
		t.Errorf("a carriage return reached the prose: %q", got.Body)
	}
	if got.Body != "# Entropy\n\nA measure of disorder.\n" {
		t.Errorf("the prose is not what was written: %q", got.Body)
	}

	onDisk, err := os.ReadFile(filepath.Join(v.Path, "Entropy.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != raw {
		t.Fatalf("reading changed the file\n want %q\n  got %q", raw, string(onDisk))
	}

	// What the index is built from is the file, so every offset into the body
	// it holds is an offset into those bytes.
	indexed := markdown.Parse(got.Fingerprint, onDisk)
	if !strings.Contains(indexed.Body, "\r\n") {
		t.Errorf("the indexed body lost the file's line endings: %q", indexed.Body)
	}
	if !bytes.HasSuffix(onDisk, []byte(indexed.Body)) {
		t.Errorf("the indexed body is not the tail of the file: %q", indexed.Body)
	}
	if strings.Index(indexed.Body, "A measure") == strings.Index(got.Body, "A measure") {
		t.Error("the two bodies agree on offsets, so this note has nothing to say")
	}
}

// One database holds every vault and one folder is read at a time. A read is
// answered by the vault it was given and by no other.
func TestAReadStaysInTheVaultItWasGiven(t *testing.T) {
	t.Parallel()
	u, physics := readable(t, map[string]string{
		"Entropy.md":  "# Entropy\n\nA measure of disorder.\n",
		"Momentum.md": "# Momentum\n",
	})
	_, kitchen := readable(t, map[string]string{
		"Entropy.md": "# Saffron\n\nRice, butter, cardamom.\n",
	})

	fromPhysics := read(t, u, physics, "Entropy.md")
	if !strings.Contains(fromPhysics.Body, "disorder") || strings.Contains(fromPhysics.Body, "cardamom") {
		t.Errorf("the other vault answered: %q", fromPhysics.Body)
	}
	fromKitchen := read(t, u, kitchen, "Entropy.md")
	if !strings.Contains(fromKitchen.Body, "cardamom") || strings.Contains(fromKitchen.Body, "disorder") {
		t.Errorf("the other vault answered: %q", fromKitchen.Body)
	}
	if got := read(t, u, kitchen, "Momentum.md"); got.Outcome != note.Missing {
		t.Errorf("a note of the other vault was found here: %+v", got)
	}
}
