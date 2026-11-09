package hostvault

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

func riverFiles() map[string]string {
	return map[string]string{
		"Rivers.txt":      "the Danube flows east",
		"Nandu.txt":       "Ñandú runs along the Paraná",
		"gauge.log":       strings.Repeat("level 7\n", 2500),
		"sonar.bin":       "ab\xff\xfecd",
		"notes/Danube.md": "# Danube\n",
	}
}

func TestReadRangeReturnsTheTextOfTheRangeAndTheFileStat(t *testing.T) {
	access, _ := diskVault(t, riverFiles())

	got, err := access.ReadRange(t.Context(), "Rivers.txt", 4, 6)
	if err != nil {
		t.Fatal(err)
	}

	if got.Text != "Danube" || got.Start != 4 {
		t.Errorf("the range is %q at %d, want \"Danube\" at 4", got.Text, got.Start)
	}
	if got.Stat.Path != "Rivers.txt" || got.Stat.Size != int64(len("the Danube flows east")) || got.Stat.ModTime.IsZero() {
		t.Errorf("the file stat is %+v, want the file stat of Rivers.txt with its size", got.Stat)
	}
}

func TestReadRangeMovesStartInsideCharacterToNextCharacter(t *testing.T) {
	access, _ := diskVault(t, riverFiles())

	got, err := access.ReadRange(t.Context(), "Nandu.txt", 1, 4)
	if err != nil {
		t.Fatal(err)
	}

	if got.Start != 2 || got.Text != "and" {
		t.Errorf("the range is %q at %d, want \"and\" at 2", got.Text, got.Start)
	}
}

func TestReadRangeOmitsACharacterTheRangeEndsInside(t *testing.T) {
	access, _ := diskVault(t, riverFiles())

	// "Ñandú" is Ñ (2 bytes), a, n, d, ú (2 bytes): the four bytes from 2 end inside ú.
	got, err := access.ReadRange(t.Context(), "Nandu.txt", 2, 4)
	if err != nil {
		t.Fatal(err)
	}

	if got.Start != 2 || got.Text != "and" {
		t.Errorf("the range is %q at %d, want \"and\" at 2", got.Text, got.Start)
	}
}

func TestReadRangeOfZeroOrOverMaxReads16000Bytes(t *testing.T) {
	access, _ := diskVault(t, riverFiles())

	for _, length := range []int{0, 16000, 16001, 50000} {
		got, err := access.ReadRange(t.Context(), "gauge.log", 0, length)
		if err != nil {
			t.Errorf("a read of %d bytes failed: %v", length, err)
			continue
		}
		if len(got.Text) != 16000 || got.Start != 0 || got.Stat.Size != 20000 {
			t.Errorf("a read of %d bytes returned %d bytes at %d of a file of %d, want 16000 at 0 of 20000", length, len(got.Text), got.Start, got.Stat.Size)
		}
	}
}

func TestReadRangeOfBytesThatAreNotUTF8IsNotText(t *testing.T) {
	access, _ := diskVault(t, riverFiles())

	_, err := access.ReadRange(t.Context(), "sonar.bin", 0, 6)

	if !errors.Is(err, pluginport.ErrNotText) {
		t.Errorf("a range of bytes that are not UTF-8 was read with %v, want ErrNotText", err)
	}
	if got, err := access.ReadRange(t.Context(), "sonar.bin", 4, 2); err != nil || got.Text != "cd" {
		t.Errorf("a range of UTF-8 bytes of the same file was read as %q (%v), want \"cd\"", got.Text, err)
	}
}

func TestReadRangeOfAFolderOrAMissingPathIsNotExist(t *testing.T) {
	access, _ := diskVault(t, riverFiles())

	for _, path := range []string{"notes", "Volga.txt", "notes/Volga.md"} {
		if _, err := access.ReadRange(t.Context(), path, 0, 10); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("a ranged read of %s was %v, want fs.ErrNotExist", path, err)
		}
	}
}

func TestReadRangeWithoutAVaultReturnsErrNoVault(t *testing.T) {
	access := Vault{Current: func() (domain.Vault, bool) { return domain.Vault{}, false }}

	if _, err := access.ReadRange(t.Context(), "Rivers.txt", 0, 10); !errors.Is(err, ErrNoVault) {
		t.Errorf("a ranged read with no vault was %v, want ErrNoVault", err)
	}
}
