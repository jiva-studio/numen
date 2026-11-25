package identityseed_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/tools/syncserver/identityseed"
)

func writeSeed(t *testing.T) string {
	t.Helper()
	seed, err := identityseed.Generate()
	if err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	if err := identityseed.Write(folder, seed); err != nil {
		t.Fatal(err)
	}
	return folder
}

func TestCheckAcceptsWhatGenerateWrote(t *testing.T) {
	if err := identityseed.Check(writeSeed(t)); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateKeepsTheAccountOfTheWords(t *testing.T) {
	first, err := identityseed.Generate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := identityseed.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if first.Identity.Fingerprint != second.Identity.Fingerprint {
		t.Fatalf("the account changed between runs: %s and %s", first.Identity.Fingerprint, second.Identity.Fingerprint)
	}
}

func TestCheckRefusesAServerStateOfAnotherLayout(t *testing.T) {
	folder := writeSeed(t)
	if err := os.WriteFile(filepath.Join(folder, identityseed.ServerFile), []byte(`{"format":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := identityseed.Check(folder); !errors.Is(err, identityseed.ErrOutOfDate) {
		t.Fatalf("got %v, want ErrOutOfDate", err)
	}
}

func TestCheckRefusesMissingFiles(t *testing.T) {
	if err := identityseed.Check(t.TempDir()); !errors.Is(err, identityseed.ErrOutOfDate) {
		t.Fatalf("got %v, want ErrOutOfDate", err)
	}
}
