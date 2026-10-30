package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// session is one installation: its own registry and index, so a test never
// touches the machine the tests run on.
type session struct {
	t     *testing.T
	vault string
	base  []string
	// bin is where an erased folder goes. A test sends nothing to the trash of
	// the machine it runs on.
	bin *fakeTrash
	// said is what the commands run here wrote beside their answers.
	reply bytes.Buffer
}

func newSession(t *testing.T) *session {
	t.Helper()
	dir := t.TempDir()
	return &session{
		t:     t,
		vault: testsupport.CopyVault(t),
		bin:   &fakeTrash{into: t.TempDir()},
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
	err := cli.Run(context.Background(), &out, &s.reply, append(s.base, args...), s.deps)
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

// addVault is a second vault of this installation, with an identity of its own.
// An installation keeps a vault, so a test that forgets or erases one has two.
func (s *session) addVault(name string) string {
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
	testsupport.Close(s.t, write)
	was := os.Stdin
	os.Stdin = read
	s.t.Cleanup(func() { os.Stdin = was })
	testsupport.CloseAfter(s.t, read)
}

// fakeTrash is a trash of this test's own: it moves what it is given into a folder the
// test made, and answers refuse where that is set.
type fakeTrash struct {
	into   string
	refuse error
	took   []string
}

func (b *fakeTrash) Trash(path string) error {
	if b.refuse != nil {
		return fmt.Errorf("%s: %w", path, b.refuse)
	}
	b.took = append(b.took, path)
	return os.Rename(path, filepath.Join(b.into, filepath.Base(path)))
}

// trash is the bin an erased folder goes to, answering refuse where one is
// given.
func (s *session) trash(refuse error) *fakeTrash {
	s.t.Helper()
	s.bin.refuse = refuse
	return s.bin
}

// getListMarks is the two characters vault list puts before a vault's name.
func getListMarks(listing, name string) string {
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
	// vault and not a folder.
	if _, err := os.Stat(filepath.Join(s.vault, ".numen", "config.json")); err != nil {
		t.Errorf("vault has no identity after being added: %v", err)
	}

	items := s.mustRun("vault", "list")
	if !strings.Contains(items, "demo") {
		t.Errorf("vault list said:\n%s", items)
	}

	scanned := s.mustRun("scan", "demo")
	if !strings.Contains(scanned, "index now holds 14 notes") {
		t.Errorf("scan said:\n%s", scanned)
	}

	found := s.mustRun("search", "demo", "entropy")
	if !strings.Contains(found, "Entropy") {
		t.Errorf("search said:\n%s", found)
	}

	// An installation with no model searches by words alone and says nothing
	// about it: half a search is a whole answer.
	if s.reply.Len() > 0 {
		t.Errorf("the commands said beside their answers:\n%s", s.reply.String())
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

func TestUnknownVaultReturnsWhatToDo(t *testing.T) {
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

func TestAddingAFileInsteadOfAFolderFails(t *testing.T) {
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
	if parseIdentity(first) != parseIdentity(second) {
		t.Errorf("identity changed on re-adding:\n%s\n%s", first, second)
	}
}

func parseIdentity(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if id, ok := strings.CutPrefix(strings.TrimSpace(line), "id "); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

func TestACopiedVaultIsRefusedInsteadOfIndexed(t *testing.T) {
	// Copying a vault folder copies its identity, and two folders claiming one
	// identity cannot be told apart. Indexing either would write one vault's
	// notes under the other's rows, so adding the second fails and the
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
	items := s.mustRun("vault", "list")
	if strings.Count(items, s.vault) != 1 || strings.Contains(items, copied) {
		t.Errorf("vault list after the rejection:\n%s", items)
	}
}

func TestAMovedVaultIsRecognisedInsteadOfRefused(t *testing.T) {
	// Identity does not depend on the path precisely so that a folder can move.
	// The registry is what has to catch up.
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "moved")
	s.mustRun("scan", "moved")
	was := readIdentity(t, s.vault)

	moved := filepath.Join(filepath.Dir(s.vault), "somewhere-else")
	if err := os.Rename(s.vault, moved); err != nil {
		t.Fatal(err)
	}

	out, err := s.run("vault", "add", moved, "--name", "moved")
	if err != nil {
		t.Fatalf("a moved vault failed: %v\n%s", err, out)
	}

	items := s.mustRun("vault", "list")
	if strings.Count(items, "moved") != 1 {
		t.Errorf("the move produced a second entry:\n%s", items)
	}
	if !strings.Contains(items, moved) {
		t.Errorf("the registry still points at the old location:\n%s", items)
	}

	// The identity travelled with the folder, so the index it already built is
	// still the right one.
	if now := readIdentity(t, moved); now != was {
		t.Errorf("the moved vault is %s, and was %s", now, was)
	}
}

// readIdentity is what a vault calls itself, read out of the folder.
func readIdentity(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(path, ".numen", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var target struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &target); err != nil {
		t.Fatal(err)
	}
	if target.ID == "" {
		t.Fatalf("the vault at %s calls itself nothing", path)
	}
	return target.ID
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
	// A link that answers to nothing says so and not being left out.
	broken := s.mustRun("links", "demo", "edge/broken-links.md")
	if !strings.Contains(broken, "nothing by that name") {
		t.Errorf("a dangling link was not reported:\n%s", broken)
	}
}

func TestRenamingPreservesTheFolderWhereItIs(t *testing.T) {
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

	items := s.mustRun("vault", "list")
	if !strings.Contains(items, "after") || strings.Contains(items, "before") {
		t.Errorf("vault list after the rename:\n%s", items)
	}
}

func TestForgettingKeepsTheFolderAndReturnsSo(t *testing.T) {
	s := newSession(t)
	added := s.mustRun("vault", "add", s.vault, "--name", "leaving")
	s.addVault("staying")

	out := s.mustRun("vault", "remove", "leaving")
	if !strings.Contains(out, s.vault) || !strings.Contains(out, "the folder is still") {
		t.Errorf("vault remove said:\n%s", out)
	}
	if _, err := os.Stat(s.vault); err != nil {
		t.Errorf("the folder went with the entry: %v", err)
	}
	if items := s.mustRun("vault", "list"); strings.Contains(items, "leaving") {
		t.Errorf("the vault is still on the list:\n%s", items)
	}

	// The identity stayed in the folder, so the vault that comes back is the one
	// that left.
	again := s.mustRun("vault", "add", s.vault, "--name", "leaving")
	if parseIdentity(again) != parseIdentity(added) {
		t.Errorf("adding it again made another vault:\n%s\n%s", added, again)
	}
}

func TestErasingAsksBeforeItActs(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "asked")
	s.addVault("other")
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
	if items := s.mustRun("vault", "list"); !strings.Contains(items, "asked") {
		t.Errorf("the vault left the list:\n%s", items)
	}
}

func TestErasingWithYesTrashesTheFolderAndForgetsTheVault(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "erased")
	s.addVault("other")
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
	if items := s.mustRun("vault", "list"); strings.Contains(items, "erased") {
		t.Errorf("the vault is still on the list:\n%s", items)
	}
}

func TestAMachineWithNowhereToInsertItDeletesNothing(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "kept")
	s.addVault("other")
	s.trash(machineport.ErrNoTrash)

	out, err := s.run("vault", "erase", "kept", "--yes")
	if !errors.Is(err, machineport.ErrNoTrash) {
		t.Fatalf("erasing gave %v, want %v\n%s", err, machineport.ErrNoTrash, out)
	}
	if _, err := os.Stat(s.vault); err != nil {
		t.Errorf("the folder went with nowhere to put it: %v", err)
	}
	if items := s.mustRun("vault", "list"); !strings.Contains(items, "kept") {
		t.Errorf("the vault left the list:\n%s", items)
	}
}

func TestOpenRecordsTheVaultTheNextWindowOpens(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "first")
	s.addVault("second")

	if mark := getListMarks(s.mustRun("vault", "list"), "second"); mark != "  " {
		t.Errorf("a vault is marked %q before any was opened", mark)
	}

	out := s.mustRun("vault", "open", "second")
	if !strings.Contains(out, "second") {
		t.Errorf("vault open said:\n%s", out)
	}

	items := s.mustRun("vault", "list")
	if mark := getListMarks(items, "second"); mark != "* " {
		t.Errorf("the vault opened last is marked %q:\n%s", mark, items)
	}
	if mark := getListMarks(items, "first"); mark != "  " {
		t.Errorf("a vault that was not opened is marked %q:\n%s", mark, items)
	}

	// The mark for a folder that is not there is its own, and a vault can carry
	// both.
	if err := os.Rename(s.vault, filepath.Join(filepath.Dir(s.vault), "elsewhere")); err != nil {
		t.Fatal(err)
	}
	if mark := getListMarks(s.mustRun("vault", "list"), "first"); mark != " ?" {
		t.Errorf("a vault whose folder is gone is marked %q", mark)
	}
}

func TestAVaultThatMatchesNothingNamesWhatWasTyped(t *testing.T) {
	s := newSession(t)
	s.mustRun("vault", "add", s.vault, "--name", "here")
	for _, args := range [][]string{
		{"vault", "rename", "nowhere", "elsewhere"},
		{"vault", "remove", "nowhere"},
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
		{"vault", "remove", "single"},
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
	if items := s.mustRun("vault", "list"); !strings.Contains(items, "single") {
		t.Errorf("the vault left the list:\n%s", items)
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
