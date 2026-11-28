package note_test

import (
	"bytes"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
)

func getVaultKey() [32]byte {
	var key [32]byte
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func getReferencePathHash(tb testing.TB, key [32]byte, folded string) []byte {
	tb.Helper()
	derived, err := hkdf.Key(sha256.New, key[:], nil, "numen-sync/path-hash/v1", 32)
	if err != nil {
		tb.Fatal(err)
	}
	mac := hmac.New(sha256.New, derived)
	mac.Write([]byte(folded))
	return mac.Sum(nil)
}

func TestHash_IsTheHexSHA256OfTheContent(t *testing.T) {
	sum := sha256.Sum256([]byte("milk"))

	if got := note.Hash([]byte("milk")); got != hex.EncodeToString(sum[:]) {
		t.Errorf("Hash = %s, want %x", got, sum)
	}
}

func TestNormalizePath_ComposesCharactersToNFC(t *testing.T) {
	decomposed := "Café/menu.md"

	got, err := note.NormalizePath(decomposed)
	if err != nil {
		t.Fatal(err)
	}

	if got != "Café/menu.md" {
		t.Errorf("NormalizePath = %q, want the composed form %q", got, "Café/menu.md")
	}
}

func TestNormalizePath_KeepsTheCaseOfThePath(t *testing.T) {
	got, err := note.NormalizePath("Food/Shopping list.md")
	if err != nil {
		t.Fatal(err)
	}

	if got != "Food/Shopping list.md" {
		t.Errorf("NormalizePath = %q, want the path as written", got)
	}
}

func TestNormalizePath_RejectsAPathOutsideTheVault(t *testing.T) {
	for _, path := range []string{"", "/Shopping list.md", "../Shopping list.md", "Food/../../Shopping list.md"} {
		if _, err := note.NormalizePath(path); !errors.Is(err, note.ErrInvalidPath) {
			t.Errorf("NormalizePath(%q) error = %v, want ErrInvalidPath", path, err)
		}
	}
}

func TestPathHash_IsAKeyedHashOfTheFoldedPath(t *testing.T) {
	key := getVaultKey()

	got := note.HashPath(key, "food/shopping list.md")

	want := getReferencePathHash(t, key, "food/shopping list.md")
	if !bytes.Equal(got, want) {
		t.Errorf("HashPath = %x, want HMAC-SHA256 of the folded path under HKDF(vault key) = %x", got, want)
	}
}

func TestPathHash_IgnoresCaseAndCompositionOfThePath(t *testing.T) {
	key := getVaultKey()

	lower := note.HashPath(key, "café/menu.md")

	for _, path := range []string{"Café/Menu.md", "Café/MENU.md"} {
		if got := note.HashPath(key, path); !bytes.Equal(got, lower) {
			t.Errorf("HashPath(%q) = %x, want %x of the same path in lower case", path, got, lower)
		}
	}
}

func TestPathHash_DiffersPerVaultKeyAndPerPath(t *testing.T) {
	key := getVaultKey()
	other := getVaultKey()
	other[0] ^= 1

	if bytes.Equal(note.HashPath(key, "a.md"), note.HashPath(other, "a.md")) {
		t.Error("two vault keys give one path hash for a path, want different hashes")
	}
	if bytes.Equal(note.HashPath(key, "a.md"), note.HashPath(key, "b.md")) {
		t.Error("two paths give one path hash, want different hashes")
	}
}

func TestDecide_AdoptsTheHeadWhenTheLocalNoteEqualsIt(t *testing.T) {
	if got := note.Decide("h2", "h1", "h2"); got != note.Adopt {
		t.Errorf("Decide(local = head) = %v, want Adopt", got)
	}
	if got := note.Decide("h2", "", "h2"); got != note.Adopt {
		t.Errorf("Decide(local = head, no base) = %v, want Adopt", got)
	}
}

func TestDecide_DownloadsWhenTheLocalNoteIsAbsentOrEqualsTheBase(t *testing.T) {
	if got := note.Decide("", "", "h2"); got != note.Download {
		t.Errorf("Decide(absent local) = %v, want Download", got)
	}
	if got := note.Decide("", "h1", "h2"); got != note.Download {
		t.Errorf("Decide(absent local, known base) = %v, want Download", got)
	}
	if got := note.Decide("h1", "h1", "h2"); got != note.Download {
		t.Errorf("Decide(local = base) = %v, want Download", got)
	}
}

func TestDecide_ConflictsWhenTheLocalNoteDiffersFromTheBaseAndTheHead(t *testing.T) {
	if got := note.Decide("h3", "h1", "h2"); got != note.Conflict {
		t.Errorf("Decide(local differs from both) = %v, want Conflict", got)
	}
	if got := note.Decide("h3", "", "h2"); got != note.Conflict {
		t.Errorf("Decide(a local note with no base) = %v, want Conflict", got)
	}
}

func TestSealMeta_PadsTheSealedMetadataToAMultipleOf64Bytes(t *testing.T) {
	t.Parallel()
	key, vaultID, fileID := getVaultKey(), []byte("vault"), []byte("file")
	for _, path := range []string{"a.md", "notes/a longer name.md", "notes/projects/2026/a much longer name of a note.md"} {
		sealed, err := note.SealMeta(bytes.NewReader(bytes.Repeat([]byte{7}, 24)), key, vaultID, fileID, note.Meta{Path: path, Key: make([]byte, 32), Hash: "h"})
		if err != nil {
			t.Fatal(err)
		}
		if body := len(sealed) - 1 - 24 - 16; body%64 != 0 {
			t.Errorf("%q: sealed body has %d bytes, want a multiple of 64", path, body)
		}
		got, err := note.OpenMeta(key, vaultID, fileID, sealed)
		if err != nil || got.Path != path {
			t.Errorf("%q: OpenMeta() = %+v, %v, want the path back", path, got, err)
		}
	}
}
