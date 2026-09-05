package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// session is one installation: its own registry and index, so a test never
// touches the machine the tests run on.
type session struct {
	t     *testing.T
	vault string
	base  []string
	// said is what the commands run here wrote beside their answers.
	said bytes.Buffer
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
	// No embedder and nothing this machine supplies: a test must not reach a
	// model, a service, an account or a process.
	err := cli.Run(context.Background(), &out, &s.said, append(s.base, args...),
		container.Config{}.Indexing(settings.Indexing{}))
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

// another is a second vault of this installation, with an identity of its own.
// An installation keeps a vault, so a test that forgets or erases one has two.
func (s *session) another(name string) string {
	s.t.Helper()
	dir := testsupport.TempDir(s.t)
	s.mustRun("vault", "add", dir, "--name", name)
	return dir
}

// types is what the person answers when a command asks.
func (s *session) types(answer string) {
	s.t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		s.t.Fatal(err)
	}
	if _, err := io.WriteString(write, answer); err != nil {
		s.t.Fatal(err)
	}
	write.Close()
	was := os.Stdin
	os.Stdin = read
	s.t.Cleanup(func() { os.Stdin = was; read.Close() })
}

// bin is a trash of this test's own: it moves what it is given into a folder the
// test made, and answers refuse where that is set.
type bin struct {
	into   string
	refuse error
	took   []string
}

func (b *bin) Trash(path string) error {
	if b.refuse != nil {
		return fmt.Errorf("%s: %w", path, b.refuse)
	}
	b.took = append(b.took, path)
	return os.Rename(path, filepath.Join(b.into, filepath.Base(path)))
}

// trash puts a bin where an erased folder goes, for as long as the test runs.
func (s *session) trash(refuse error) *bin {
	s.t.Helper()
	b := &bin{into: s.t.TempDir(), refuse: refuse}
	s.t.Cleanup(cli.ErasesInto(b))
	return b
}

// marks is the two characters vault list puts before a vault's name.
func marks(listing, name string) string {
	for _, line := range strings.Split(listing, "\n") {
		if len(line) > 2 && strings.HasPrefix(strings.TrimLeft(line, " *?"), name) {
			return line[:2]
		}
	}
	return ""
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
	if !strings.Contains(scanned, "14 notes: 14 indexed") {
		t.Errorf("scan said:\n%s", scanned)
	}

	found := s.mustRun("search", "demo", "entropy")
	if !strings.Contains(found, "Entropy") {
		t.Errorf("search said:\n%s", found)
	}

	// An installation with no model searches by words alone and says nothing
	// about it: half a search is a whole answer.
	if s.said.Len() > 0 {
		t.Errorf("the commands said beside their answers:\n%s", s.said.String())
	}
}

func TestSecondScanChangesNothing(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "demo")
	s.mustRun("scan", "demo")

	again := s.mustRun("scan", "demo")
	if !strings.Contains(again, "0 indexed") || !strings.Contains(again, "14 unchanged") {
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

func TestRenamingLeavesTheFolderWhereItIs(t *testing.T) {
	// The name is what a person calls the collection; the folder keeps the name
	// the filesystem gives it.
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "before")

	out := s.mustRun("vault", "rename", "before", "after")
	if !strings.Contains(out, "after") || !strings.Contains(out, s.vault) {
		t.Errorf("vault rename said:\n%s", out)
	}
	if _, err := os.Stat(s.vault); err != nil {
		t.Errorf("the folder is not where it was: %v", err)
	}

	listed := s.mustRun("vault", "list")
	if !strings.Contains(listed, "after") || strings.Contains(listed, "before") {
		t.Errorf("vault list after the rename:\n%s", listed)
	}
}

func TestForgettingKeepsTheFolderAndSaysSo(t *testing.T) {
	s := newSession(t)
	added := s.mustRun("vault", "add", s.vault, "--name", "leaving")
	s.another("staying")

	out := s.mustRun("vault", "forget", "leaving")
	if !strings.Contains(out, s.vault) || !strings.Contains(out, "the folder is still") {
		t.Errorf("vault forget said:\n%s", out)
	}
	if _, err := os.Stat(s.vault); err != nil {
		t.Errorf("the folder went with the entry: %v", err)
	}
	if listed := s.mustRun("vault", "list"); strings.Contains(listed, "leaving") {
		t.Errorf("the vault is still on the list:\n%s", listed)
	}

	// The identity stayed in the folder, so the vault that comes back is the one
	// that left.
	again := s.mustRun("vault", "add", s.vault, "--name", "leaving")
	if identity(again) != identity(added) {
		t.Errorf("adding it again made another vault:\n%s\n%s", added, again)
	}
}

func TestErasingAsksBeforeItActs(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "asked")
	s.another("other")
	b := s.trash(nil)
	s.types("\n")

	out, err := s.run("vault", "erase", "asked")
	if err == nil {
		t.Fatalf("an unanswered erase went ahead:\n%s", out)
	}
	// What will happen, and what folder it is.
	for _, want := range []string{"trash", s.vault} {
		if !strings.Contains(out, want) {
			t.Errorf("the question does not mention %q:\n%s", want, out)
		}
	}
	if len(b.took) != 0 {
		t.Errorf("a folder went to the trash unasked: %v", b.took)
	}
	if _, err := os.Stat(s.vault); err != nil {
		t.Errorf("the folder went: %v", err)
	}
	if listed := s.mustRun("vault", "list"); !strings.Contains(listed, "asked") {
		t.Errorf("the vault left the list:\n%s", listed)
	}
}

func TestErasingWithYesTrashesTheFolderAndForgetsTheVault(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "erased")
	s.another("other")
	b := s.trash(nil)

	out := s.mustRun("vault", "erase", "erased", "--yes")
	if !strings.Contains(out, "erased") {
		t.Errorf("vault erase said:\n%s", out)
	}
	if len(b.took) != 1 || b.took[0] != s.vault {
		t.Errorf("what went to the trash was %v, want %s", b.took, s.vault)
	}
	if _, err := os.Stat(s.vault); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the folder is still at %s: %v", s.vault, err)
	}
	if listed := s.mustRun("vault", "list"); strings.Contains(listed, "erased") {
		t.Errorf("the vault is still on the list:\n%s", listed)
	}
}

func TestAMachineWithNowhereToPutItDeletesNothing(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "kept")
	s.another("other")
	s.trash(port.ErrNoTrash)

	out, err := s.run("vault", "erase", "kept", "--yes")
	if !errors.Is(err, port.ErrNoTrash) {
		t.Fatalf("erasing gave %v, want %v\n%s", err, port.ErrNoTrash, out)
	}
	if _, err := os.Stat(s.vault); err != nil {
		t.Errorf("the folder went with nowhere to put it: %v", err)
	}
	if listed := s.mustRun("vault", "list"); !strings.Contains(listed, "kept") {
		t.Errorf("the vault left the list:\n%s", listed)
	}
}

func TestOpenRecordsTheVaultTheNextWindowOpens(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "first")
	s.another("second")

	if mark := marks(s.mustRun("vault", "list"), "second"); mark != "  " {
		t.Errorf("a vault is marked %q before any was opened", mark)
	}

	out := s.mustRun("vault", "open", "second")
	if !strings.Contains(out, "second") {
		t.Errorf("vault open said:\n%s", out)
	}

	listed := s.mustRun("vault", "list")
	if mark := marks(listed, "second"); mark != "* " {
		t.Errorf("the vault opened last is marked %q:\n%s", mark, listed)
	}
	if mark := marks(listed, "first"); mark != "  " {
		t.Errorf("a vault that was not opened is marked %q:\n%s", mark, listed)
	}

	// The mark for a folder that is not there is its own, and a vault can carry
	// both.
	if err := os.Rename(s.vault, filepath.Join(filepath.Dir(s.vault), "elsewhere")); err != nil {
		t.Fatal(err)
	}
	if mark := marks(s.mustRun("vault", "list"), "first"); mark != " ?" {
		t.Errorf("a vault whose folder is gone is marked %q", mark)
	}
}

func TestAVaultThatMatchesNothingNamesWhatWasTyped(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "here")
	for _, args := range [][]string{
		{"vault", "rename", "nowhere", "elsewhere"},
		{"vault", "forget", "nowhere"},
		{"vault", "erase", "nowhere", "--yes"},
		{"vault", "open", "nowhere"},
	} {
		out, err := s.run(args...)
		if err == nil {
			t.Errorf("numen-cli %s succeeded:\n%s", strings.Join(args, " "), out)
			continue
		}
		if !strings.Contains(err.Error(), "nowhere") {
			t.Errorf("numen-cli %s: the error does not name what was typed: %v",
				strings.Join(args, " "), err)
		}
	}
}

func TestTheOnlyVaultAnInstallationHasStays(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "single")
	b := s.trash(nil)

	for _, args := range [][]string{
		{"vault", "forget", "single"},
		{"vault", "erase", "single", "--yes"},
	} {
		out, err := s.run(args...)
		if !errors.Is(err, vaults.ErrLastVault) {
			t.Errorf("numen-cli %s gave %v, want %v\n%s",
				strings.Join(args, " "), err, vaults.ErrLastVault, out)
		}
	}
	if len(b.took) != 0 {
		t.Errorf("a folder went to the trash: %v", b.took)
	}
	if _, err := os.Stat(s.vault); err != nil {
		t.Errorf("the folder went: %v", err)
	}
	if listed := s.mustRun("vault", "list"); !strings.Contains(listed, "single") {
		t.Errorf("the vault left the list:\n%s", listed)
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
