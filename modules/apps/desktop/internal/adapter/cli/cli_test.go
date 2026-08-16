package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/cli"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// session is one installation: its own registry and index, so a test never
// touches the machine the tests run on.
type session struct {
	t     *testing.T
	vault string
	base  []string
}

func newSession(t *testing.T) *session {
	t.Helper()
	dir := t.TempDir()
	return &session{
		t:     t,
		vault: testsupport.CopyVault(t),
		base: []string{
			"--registry", filepath.Join(dir, "vaults.json"),
			"--index", filepath.Join(dir, "index.db"),
		},
	}
}

func (s *session) run(args ...string) (string, error) {
	s.t.Helper()
	var out bytes.Buffer
	err := cli.Run(context.Background(), &out, append(s.base, args...))
	return out.String(), err
}

func (s *session) mustRun(args ...string) string {
	s.t.Helper()
	out, err := s.run(args...)
	if err != nil {
		s.t.Fatalf("numen-cli %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func TestAddScanSearch(t *testing.T) {
	s := newSession(t)

	added := s.mustRun("vault", "add", s.vault, "--name", "demo")
	if !strings.Contains(added, "added demo") {
		t.Errorf("vault add said:\n%s", added)
	}

	// The identity is written into the vault, which is what makes the folder a
	// vault rather than a folder.
	if _, err := os.Stat(filepath.Join(s.vault, ".numen", "config.json")); err != nil {
		t.Errorf("vault has no identity after being added: %v", err)
	}

	listed := s.mustRun("vault", "list")
	if !strings.Contains(listed, "demo") {
		t.Errorf("vault list said:\n%s", listed)
	}

	scanned := s.mustRun("scan", "demo")
	if !strings.Contains(scanned, "8 notes: 8 indexed") {
		t.Errorf("scan said:\n%s", scanned)
	}

	found := s.mustRun("search", "demo", "entropy")
	if !strings.Contains(found, "Entropy") {
		t.Errorf("search said:\n%s", found)
	}
}

func TestSecondScanChangesNothing(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "demo")
	s.mustRun("scan", "demo")

	again := s.mustRun("scan", "demo")
	if !strings.Contains(again, "0 indexed") || !strings.Contains(again, "8 unchanged") {
		t.Errorf("rescanning an untouched vault reindexed something:\n%s", again)
	}
}

func TestFlagsAreAcceptedAfterThePath(t *testing.T) {
	// `vault add <path> --name x` is the order people type, and the standard
	// flag package stops parsing at the first positional argument.
	s := newSession(t)
	out := s.mustRun("vault", "add", s.vault, "--name", "chosen")
	if !strings.Contains(out, "added chosen") {
		t.Errorf("the name flag after the path was dropped:\n%s", out)
	}
}

func TestSearchFindsNothingWithoutFailing(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "demo")
	s.mustRun("scan", "demo")

	out := s.mustRun("search", "demo", "quagmire")
	if !strings.Contains(out, "nothing found") {
		t.Errorf("search said:\n%s", out)
	}
}

func TestUnknownVaultSaysWhatToDo(t *testing.T) {
	s := newSession(t)
	_, err := s.run("scan", "missing")
	if err == nil {
		t.Fatal("scanning an unknown vault succeeded")
	}
	if !strings.Contains(err.Error(), "numen-cli vault add") {
		t.Errorf("error does not say how to fix it: %v", err)
	}
}

func TestListBeforeAnythingIsAdded(t *testing.T) {
	s := newSession(t)
	out := s.mustRun("vault", "list")
	if !strings.Contains(out, "no vaults yet") {
		t.Errorf("vault list said:\n%s", out)
	}
}

func TestUsageIsShownWhenNothingIsAsked(t *testing.T) {
	s := newSession(t)
	out, err := s.run()
	if err == nil {
		t.Error("running with no command reported success")
	}
	if !strings.Contains(out, "usage:") {
		t.Errorf("no usage was printed:\n%s", out)
	}
}

func TestUnknownCommandIsRejected(t *testing.T) {
	s := newSession(t)
	if _, err := s.run("frobnicate"); err == nil {
		t.Error("an unknown command was accepted")
	}
	if _, err := s.run("vault", "frobnicate"); err == nil {
		t.Error("an unknown vault subcommand was accepted")
	}
}

func TestAddingAFileRatherThanAFolderFails(t *testing.T) {
	s := newSession(t)
	_, err := s.run("vault", "add", filepath.Join(s.vault, "Thermodynamics.md"))
	if err == nil {
		t.Error("a file was accepted as a vault")
	}
}

func TestReaddingAVaultKeepsItsIdentity(t *testing.T) {
	s := newSession(t)
	first := s.mustRun("vault", "add", s.vault, "--name", "demo")
	second := s.mustRun("vault", "add", s.vault, "--name", "demo")
	if identity(first) != identity(second) {
		t.Errorf("identity changed on re-adding:\n%s\n%s", first, second)
	}
}

func identity(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if id, ok := strings.CutPrefix(strings.TrimSpace(line), "id "); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func TestACopiedVaultIsRefusedRatherThanIndexed(t *testing.T) {
	// Copying a vault folder copies its identity, and two folders claiming one
	// identity cannot be told apart. Indexing either would write one vault's
	// notes under the other's rows, so adding the second is refused and the
	// error says what to do about it.
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "original")

	copied := testsupport.CopyVault(t)
	out, err := s.run("vault", "add", copied, "--name", "copy")
	if err == nil {
		t.Fatalf("a copy with the same identity was accepted:\n%s", out)
	}
	for _, want := range []string{copied, s.vault, "delete"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q: %v", want, err)
		}
	}

	// The first vault is untouched, and still the only one.
	listed := s.mustRun("vault", "list")
	if strings.Count(listed, s.vault) != 1 || strings.Contains(listed, copied) {
		t.Errorf("vault list after the refusal:\n%s", listed)
	}
}

func TestAMovedVaultIsRecognisedRatherThanRefused(t *testing.T) {
	// Identity does not depend on the path precisely so that a folder can move.
	// The registry is what has to catch up.
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "moved")
	s.mustRun("scan", "moved")

	moved := filepath.Join(filepath.Dir(s.vault), "somewhere-else")
	if err := os.Rename(s.vault, moved); err != nil {
		t.Fatal(err)
	}

	out, err := s.run("vault", "add", moved, "--name", "moved")
	if err != nil {
		t.Fatalf("a moved vault was refused: %v\n%s", err, out)
	}

	listed := s.mustRun("vault", "list")
	if strings.Count(listed, "moved") != 1 {
		t.Errorf("the move produced a second entry:\n%s", listed)
	}
	if !strings.Contains(listed, moved) {
		t.Errorf("the registry still points at the old location:\n%s", listed)
	}

	// The identity travelled with the folder, so the index it already built is
	// still the right one.
	if !strings.Contains(s.mustRun("scan", "moved"), "0 indexed") {
		t.Error("the moved vault was reindexed from scratch")
	}
}

func TestLinksShowsBothDirections(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "demo")
	s.mustRun("scan", "demo")

	out := s.mustRun("links", "demo", "notes/Entropy.md")
	// Thermodynamics names it as a child and mentions it in prose; the fixture
	// is what says so.
	// The child edge is the one that makes the hierarchy, and it is written as a
	// path — which is exactly the form a text match misses.
	if !strings.Contains(out, "child      Thermodynamics.md") {
		t.Errorf("the child edge is missing:\n%s", out)
	}

	from := s.mustRun("links", "demo", "Thermodynamics.md")
	if !strings.Contains(from, "points at:") {
		t.Errorf("links said:\n%s", from)
	}
	// A link that answers to nothing says so rather than being left out.
	broken := s.mustRun("links", "demo", "edge/broken-links.md")
	if !strings.Contains(broken, "nothing by that name") {
		t.Errorf("a dangling link was not reported:\n%s", broken)
	}
}

func TestProblemsReportsWhatWasNotGuessedAt(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "demo")
	s.mustRun("scan", "demo")

	out := s.mustRun("problems", "demo")
	// The fixture has a note with three unusable link entries and one with
	// frontmatter that will not parse. None of it stopped the scan, and none of
	// it is invisible.
	for _, want := range []string{"broken-links.md", "no role", "sideways", "broken-frontmatter.md"} {
		if !strings.Contains(out, want) {
			t.Errorf("problems did not mention %q:\n%s", want, out)
		}
	}
}
