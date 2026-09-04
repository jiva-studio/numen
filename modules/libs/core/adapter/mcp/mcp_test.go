package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/check"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// connected is the tools as an agent meets them: over a real session, through
// the protocol, rather than by calling the handlers directly.
func connected(t *testing.T, notes map[string]string) (*sdk.ClientSession, domain.Vault) {
	t.Helper()
	v, core := built(t, notes)
	return connectedTo(t, core), v
}

// connectedTo is the same session over tools a test has adjusted.
func connectedTo(t *testing.T, core mcp.Core) *sdk.ClientSession {
	t.Helper()
	return sessionOf(t, mcp.New(core))
}

// sessionOf is a session over one server, through the protocol.
func sessionOf(t *testing.T, server *sdk.Server) *sdk.ClientSession {
	t.Helper()
	here, there := sdk.NewInMemoryTransports()
	if _, err := server.Connect(t.Context(), here, nil); err != nil {
		t.Fatal(err)
	}
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), there, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// served is a vault whose tools are built but not connected, for the questions
// that are about the transport rather than about the tools.
func served(t *testing.T) (domain.Vault, mcp.Core) {
	t.Helper()
	return built(t, map[string]string{"Entropy.md": "# Entropy\n"})
}

func built(t *testing.T, notes map[string]string) (domain.Vault, mcp.Core) {
	t.Helper()

	v := testsupport.NewVault(t, notes)
	cfg := container.Config{IndexPath: indexfile.Path(t)}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	readers, writers := filesystem.VaultReaders{}, filesystem.VaultWriters{}
	scan := vaults.Scan{
		Readers: readers, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.Queries(), Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	refresh := vaults.Refresh{Readers: readers, Notes: db.Notes()}
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	queries := db.Queries()
	cutting := cfg.Cards(queries, db.Links(), index)

	core := mcp.Core{
		Cards: mcp.Cards{
			Read:        cutting.Read,
			List:        cutting.List,
			Write:       cutting.Write,
			Create:      cutting.Create,
			RenameField: cutting.Rename,
			DeckEdit:    format.OpenDeckBody,
			StencilBody: container.StencilBody,
		},

		Showing: mcp.One(v, v.Path), Readers: readers,
		Notes: mcp.Notes{
			Queries:       queries,
			Search:        search.New(db.Passages(), readers, nil, nil, nil, 0, nil),
			Neighbourhood: note.ShowNeighbourhood{Links: db.Links(), Notes: queries},
			Links:         note.ShowLinks{Links: db.Links()},
			Problems:      check.Standard(db.Problems()),
			Create: note.Create{
				Writers: writers, Names: queries, Index: index,
			},
			Write:   note.Write{Readers: readers, Writers: writers, Index: index},
			Replace: note.Replace{Readers: readers, Writers: writers, Index: index},
			Move:    note.Move{Readers: readers, Writers: writers, Links: db.Links(), Sources: db.Sources(), Index: index},
			Rename:  note.Rename{Move: note.Move{Readers: readers, Writers: writers, Links: db.Links(), Sources: db.Sources(), Index: index}},
			Remove:  note.Remove{Writers: writers, Links: db.Links(), Known: db.SourcesKnown(), Index: index},
			Linking: note.EditLinks{Readers: readers, Writers: writers, Index: index},
		},
	}
	return v, core
}

// call runs one tool and decodes what it answered.
func call[T any](t *testing.T, s *sdk.ClientSession, name string, args any) T {
	t.Helper()
	res, err := s.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("%s: %s", name, text(res))
	}
	var out T
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return out
}

// fingerprint is what a note is at this moment, as note_read gives it. Every
// tool that writes a note takes one.
func fingerprint(t *testing.T, s *sdk.ClientSession, path string) string {
	t.Helper()
	read := call[struct {
		Notes []struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"notes"`
	}](t, s, "note_read", map[string]any{"paths": []string{path}})
	if len(read.Notes) != 1 {
		t.Fatalf("note_read answered with %d notes for %s", len(read.Notes), path)
	}
	return read.Notes[0].Fingerprint
}

// deckFingerprint is what a deck is at this moment, as card_read gives it.
// Every tool that writes a deck takes one.
func deckFingerprint(t *testing.T, s *sdk.ClientSession, path string) string {
	t.Helper()
	return call[struct {
		Fingerprint string `json:"fingerprint"`
	}](t, s, "card_read", map[string]any{"path": path}).Fingerprint
}

func failing(t *testing.T, s *sdk.ClientSession, name string, args any) string {
	t.Helper()
	res, err := s.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if !res.IsError {
		t.Fatalf("%s: expected a refusal, got %v", name, res.StructuredContent)
	}
	return text(res)
}

func text(res *sdk.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if t, ok := c.(*sdk.TextContent); ok {
			b.WriteString(t.Text)
		}
	}
	return b.String()
}

// An agent is told where the vault is once, so that no answer has to repeat it.
func TestTheVaultIsLocatedInTheInstructions(t *testing.T) {
	session, v := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})
	if got := session.InitializeResult().Instructions; !strings.Contains(got, v.Path) {
		t.Errorf("the vault's folder is not in the instructions:\n%s", got)
	}
}

// The tools an agent is given say what they take; the instructions say how a
// book is asked, which is what nothing about one tool's arguments can say.
func TestHowABookIsAskedIsInTheInstructions(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})
	said := session.InitializeResult().Instructions
	for _, rule := range []string{"source_focus", "source_read", "own words", "numen:"} {
		if !strings.Contains(said, rule) {
			t.Errorf("the instructions say nothing about %q:\n%s", rule, said)
		}
	}
}

// A note an answer speaks about is a place the person can go, and the brackets
// are the form that takes.
func TestHowANoteIsNamedInAnAnswerIsInTheInstructions(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})
	said := session.InitializeResult().Instructions
	for _, rule := range []string{"[[Harmonic oscillator]]", "[[note://<identifier>]]"} {
		if !strings.Contains(said, rule) {
			t.Errorf("the instructions say nothing about %q:\n%s", rule, said)
		}
	}
}

func TestTheToolsAreNamedForWhatTheyWorkOn(t *testing.T) {
	session, _ := connected(t, nil)
	exactly(t, serves(t, session), []string{
		"note_search", "note_titles", "note_read", "note_resolve", "note_neighbourhood",
		"note_create", "note_rewrite", "note_edit", "note_rename", "note_move", "note_remove",
		"file_read",
		"link_add", "link_update", "link_remove", "link_list",
		"card_stencil_list", "card_read", "card_add", "card_edit", "card_remove",
		"card_section_add", "card_section_rename", "card_section_remove",
		"card_deck_create", "card_stencil_create", "card_field_rename",
		"vault_get", "vault_problems",
		"source_list", "source_read", "source_recognise", "source_transcribe",
	})
}

// Both of these put megabytes of new text in the folder a person syncs, and an
// agent weighing an hour's work on their behalf is told so before it calls.
func TestReadingAndListeningSayWhatTheyWriteIntoTheVault(t *testing.T) {
	session, _ := connected(t, nil)

	for tool, area := range map[string]string{
		"source_recognise":  filesystem.OCRDir,
		"source_transcribe": filesystem.SpeechDir,
	} {
		said := describing(t, session, tool)
		for _, rule := range []string{
			"into the vault",
			filesystem.DefaultServiceDir + "/" + area,
		} {
			if !strings.Contains(said, rule) {
				t.Errorf("%s says nothing about %q:\n%s", tool, rule, said)
			}
		}
	}
}

// serves is every tool a session is offered, and each of them says what it is
// for.
func serves(t *testing.T, session *sdk.ClientSession) []string {
	t.Helper()
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	named := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		named = append(named, tool.Name)
		if tool.Description == "" {
			t.Errorf("%s has nothing to tell a model about when to use it", tool.Name)
		}
	}
	return named
}

// exactly holds a server to the tools it is for, naming what is missing and
// what is served beside them.
func exactly(t *testing.T, served, want []string) {
	t.Helper()
	beside := make(map[string]bool, len(served))
	for _, name := range served {
		beside[name] = true
	}
	for _, name := range want {
		if !beside[name] {
			t.Errorf("%s is missing", name)
		}
		delete(beside, name)
	}
	for name := range beside {
		t.Errorf("%s is served and is none of the tools this server is for", name)
	}
}

func TestANoteIsMadeAndFoundThroughTheTools(t *testing.T) {
	session, _ := connected(t, nil)

	made := created(t, session, map[string]any{
		"title": "Entropy", "body": "A measure of disorder.\n",
	})
	if len(made) != 1 || made[0].Refused != "" {
		t.Fatalf("want one note made: %+v", made)
	}
	if made[0].Path != "Entropy.md" {
		t.Fatalf("want Entropy.md, got %s", made[0].Path)
	}

	found := call[struct {
		Matches []mcp.Passage `json:"matches"`
	}](t, session, "note_search", map[string]any{"query": "disorder"})
	if len(found.Matches) != 1 || found.Matches[0].Source != made[0].Path {
		t.Errorf("a note made through the tools is not searchable: %+v", found.Matches)
	}
	if !strings.Contains(found.Matches[0].Text, "A measure of disorder.") {
		t.Errorf("a match came back without the text around it: %+v", found.Matches[0])
	}
}

// A note asks to be joined as it is made, so that it never exists unattached —
// which is the whole reason links are accepted here rather than only by
// link_add.
func TestANoteIsMadeAlreadyJoined(t *testing.T) {
	session, _ := connected(t, map[string]string{"Momentum.md": "# Momentum\n"})

	made := created(t, session, map[string]any{
		"title": "Impulse",
		"links": []map[string]any{
			{"to": "Momentum", "role": "parent", "label": "part of"},
		},
	})
	if len(made) != 1 || made[0].Refused != "" {
		t.Fatalf("want one note made: %+v", made)
	}

	links := call[struct {
		Links []mcp.Link `json:"links"`
	}](t, session, "link_list", map[string]any{"path": made[0].Path})
	if len(links.Links) != 1 {
		t.Fatalf("want the link it was made with: %+v", links.Links)
	}
	if links.Links[0].To != "Momentum.md" || links.Links[0].Role != "parent" {
		t.Errorf("want a parent link resolving to Momentum.md: %+v", links.Links[0])
	}
}

// The filesystem has no transaction over many files, so one note that cannot be
// made must not take the others down with it — nor be reported as though it
// had been made.
func TestOneNoteRefusedLeavesTheRestMade(t *testing.T) {
	session, vault := connected(t, nil)

	made := created(t, session,
		map[string]any{"title": "Impulse"},
		map[string]any{"title": "Momentum", "links": []map[string]any{
			{"to": "Impulse", "role": "nonsense"},
		}},
		map[string]any{"title": "Work"},
	)
	if len(made) != 3 {
		t.Fatalf("want an outcome for each: %+v", made)
	}
	if made[0].Refused != "" || made[2].Refused != "" {
		t.Errorf("the notes that could be made were not: %+v", made)
	}
	if made[1].Refused == "" {
		t.Fatal("a link carrying no known role was written anyway")
	}
	if made[1].Path != "" {
		t.Errorf("a note that was not made came back with a path: %+v", made[1])
	}

	// The folder is asked, not the index. A file written and then not indexed
	// is absent from the index either way, so only the disk can say whether
	// the refusal left a note behind for the person to find.
	if _, err := os.Stat(filepath.Join(vault.Path, "Momentum.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a refused note left a file in the vault: %v", err)
	}
}

// created makes each note in its own call, which is the only way the tool takes
// them. The outcomes come back in the order they were asked for.
func created(t *testing.T, session *sdk.ClientSession, notes ...map[string]any) []mcp.CreateOutcome {
	t.Helper()
	out := make([]mcp.CreateOutcome, 0, len(notes))
	for _, note := range notes {
		out = append(out, call[mcp.CreateOutcome](t, session, "note_create", note))
	}
	return out
}

// A path that names nothing is an answer, not a failure: the note may have gone
// since the agent last looked.
func TestGetSeparatesWhatIsThereFromWhatIsNot(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})

	got := call[struct {
		Notes   []mcp.Note `json:"notes"`
		Missing []string   `json:"missing"`
	}](t, session, "note_titles", map[string]any{
		"paths": []string{"Entropy.md", "gone.md"},
	})
	if len(got.Notes) != 1 || got.Notes[0].Title != "Entropy" {
		t.Errorf("want the note that is there: %+v", got.Notes)
	}
	if len(got.Missing) != 1 || got.Missing[0] != "gone.md" {
		t.Errorf("want the path that is not: %v", got.Missing)
	}
}

func TestReadingGivesBackWhatWritingWants(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})

	read := call[struct {
		Notes []mcp.Contents `json:"notes"`
	}](t, session, "note_read", map[string]any{"paths": []string{"Entropy.md"}})
	if len(read.Notes) != 1 || read.Notes[0].Fingerprint == "" {
		t.Fatalf("want one note with a fingerprint: %+v", read.Notes)
	}

	// A write answers with what the note became, so an agent writing twice has
	// what the second write needs.
	first := call[wrote](t, session, "note_rewrite", map[string]any{
		"path": "Entropy.md", "body": "# Entropy\n\nRewritten.\n",
		"fingerprint": read.Notes[0].Fingerprint,
	})
	if first.Fingerprint == "" || first.Fingerprint == read.Notes[0].Fingerprint {
		t.Fatalf("the write did not answer with the file it made: %+v", first)
	}

	second := call[wrote](t, session, "note_rewrite", map[string]any{
		"path": "Entropy.md", "body": "# Entropy\n\nAgain.\n",
		"fingerprint": first.Fingerprint,
	})
	if second.Fingerprint == "" {
		t.Errorf("the second write answered with no fingerprint: %+v", second)
	}

	again := call[struct {
		Notes []mcp.Contents `json:"notes"`
	}](t, session, "note_read", map[string]any{"paths": []string{"Entropy.md"}})
	if !strings.Contains(again.Notes[0].Body, "Again.") {
		t.Errorf("the writes did not land:\n%s", again.Notes[0].Body)
	}

	// What the read gave is two writes behind, and a write presenting it is
	// refused.
	if got := failing(t, session, "note_rewrite", map[string]any{
		"path": "Entropy.md", "body": "# Entropy\n\nOnce more.\n",
		"fingerprint": read.Notes[0].Fingerprint,
	}); !strings.Contains(got, "changed") {
		t.Errorf("want a refusal naming the change, got %q", got)
	}
}

// wrote is what note_rewrite answers with.
type wrote struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

func TestLinkingTwoNotesShowsAtBothEnds(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n",
	})

	added := added(t, session, map[string]any{
		"from": "Heat.md", "to": "Entropy", "role": "parent", "label": "follows from",
	})
	if len(added) != 1 || added[0].Refused != "" {
		t.Fatalf("want the link written: %+v", added)
	}

	around := call[struct {
		Focus   mcp.Note        `json:"focus"`
		Related []mcp.Neighbour `json:"related"`
	}](t, session, "note_neighbourhood", map[string]any{"path": "Entropy.md"})
	if len(around.Related) != 1 || around.Related[0].Seat != "child" {
		t.Errorf("the other end does not see it: %+v", around.Related)
	}

	joined := call[struct {
		Backlinks []mcp.Link `json:"backlinks"`
	}](t, session, "link_list", map[string]any{"path": "Entropy.md"})
	if len(joined.Backlinks) != 1 || joined.Backlinks[0].From != "Heat.md" {
		t.Errorf("want the note that points here: %+v", joined.Backlinks)
	}
}

// Links reaching several notes in one call are each written where they belong,
// and one that carries no known role costs only itself — not the links sharing
// a note with it.
func TestLinksGoWhereTheyBelongAndABadOneCostsOnlyItself(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n",
		"Work.md":    "# Work\n",
	})

	added := added(t, session,
		map[string]any{"from": "Heat.md", "to": "Entropy", "role": "parent"},
		map[string]any{"from": "Heat.md", "to": "Work", "role": "nonsense"},
		map[string]any{"from": "Heat.md", "to": "Work", "role": "jump"},
		map[string]any{"from": "Work.md", "to": "Entropy", "role": "ref"},
	)
	if len(added) != 4 {
		t.Fatalf("want an outcome for each: %+v", added)
	}
	if added[1].Refused == "" {
		t.Error("a link carrying no known role was written anyway")
	}
	for _, i := range []int{0, 2, 3} {
		if added[i].Refused != "" {
			t.Errorf("link %d shares a note with the bad one and was refused too: %+v", i, added[i])
		}
	}

	from := call[struct {
		Links []mcp.Link `json:"links"`
	}](t, session, "link_list", map[string]any{"path": "Heat.md"})
	if len(from.Links) != 2 {
		t.Errorf("want both good links written into Heat: %+v", from.Links)
	}

	elsewhere := call[struct {
		Links []mcp.Link `json:"links"`
	}](t, session, "link_list", map[string]any{"path": "Work.md"})
	if len(elsewhere.Links) != 1 {
		t.Errorf("want the link belonging to the other note: %+v", elsewhere.Links)
	}
}

// A note that reached the disk says where it is even when the step after the
// write did not finish. Told only that it failed, a caller writes it again and
// is refused the name it already holds.
func TestANoteOnDiskComesBackWithItsPath(t *testing.T) {
	v, core := built(t, nil)
	core.Notes.Create.Index = func(context.Context, domain.Vault, []string) error {
		return errors.New("the index is not level")
	}
	session := connectedTo(t, core)

	made := created(t, session, map[string]any{"title": "Entropy"})
	if len(made) != 1 || made[0].Refused == "" {
		t.Fatalf("want the failure reported: %+v", made)
	}
	if made[0].Path != "Entropy.md" {
		t.Errorf("the note is on disk and its path was not returned: %+v", made[0])
	}
	if _, err := os.Stat(filepath.Join(v.Path, "Entropy.md")); err != nil {
		t.Fatalf("the note this is about is not on disk: %v", err)
	}
}

// Every field of a link lands in the frontmatter, so the cap has to measure
// them. A body under the cap with an enormous label is a write over it.
func TestALinkTooLargeToWriteIsRefusedBeforeAnythingIsWritten(t *testing.T) {
	session, vault := connected(t, nil)

	huge := strings.Repeat("x", (1<<20)+1)
	if got := failing(t, session, "note_create", map[string]any{
		"title": "Entropy",
		"links": []map[string]any{{"to": "Heat", "role": "ref", "label": huge}},
	}); !strings.Contains(got, "carries at once") {
		t.Errorf("want a refusal naming the size, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(vault.Path, "Entropy.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a call refused for its size wrote a note anyway: %v", err)
	}
}

func added(t *testing.T, session *sdk.ClientSession, links ...map[string]any) []mcp.AddOutcome {
	t.Helper()
	return call[struct {
		Added []mcp.AddOutcome `json:"added"`
	}](t, session, "link_add", map[string]any{"links": links}).Added
}

func TestRemovingIsReversible(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})

	removed := call[struct {
		Removed []note.RemoveResult `json:"removed"`
	}](t, session, "note_remove", map[string]any{"paths": []string{"Entropy.md"}})
	if len(removed.Removed) != 1 || removed.Removed[0].Trashed != ".trash/Entropy.md" {
		t.Fatalf("want the note in the trash: %+v", removed.Removed)
	}

	back := call[struct {
		Notes   []mcp.Note `json:"notes"`
		Missing []string   `json:"missing"`
	}](t, session, "note_titles", map[string]any{"paths": []string{"Entropy.md"}})
	if len(back.Missing) != 1 {
		t.Errorf("a removed note is still in the index: %+v", back)
	}

}

// note_remove takes notes. A folder holds as many notes as somebody filed
// under it, and removing one by naming the folder is not what this tool does.
func TestRemovingAFolderIsRefused(t *testing.T) {
	session, v := connected(t, map[string]string{
		"Reading/Entropy.md": "# Entropy\n",
		"Reading/Order.md":   "# Order\n",
	})

	removed := call[struct {
		Removed []mcp.RemoveOutcome `json:"removed"`
	}](t, session, "note_remove", map[string]any{"paths": []string{"Reading"}})
	if len(removed.Removed) != 1 || removed.Removed[0].Refused == "" {
		t.Fatalf("a folder was not refused: %+v", removed.Removed)
	}
	if !strings.Contains(removed.Removed[0].Refused, "folder") {
		t.Errorf("the refusal reads %q", removed.Removed[0].Refused)
	}
	for _, path := range []string{"Reading/Entropy.md", "Reading/Order.md"} {
		if _, err := os.Stat(filepath.Join(v.Path, path)); err != nil {
			t.Errorf("%s went with the folder: %v", path, err)
		}
	}
}

// A ceiling that truncated in silence would read as "that is all there is".
func TestAskingForTooMuchIsRefusedRatherThanTrimmed(t *testing.T) {
	session, _ := connected(t, nil)

	paths := make([]string, 60)
	for i := range paths {
		paths[i] = "note.md"
	}
	if got := failing(t, session, "note_titles", map[string]any{"paths": paths}); !strings.Contains(got, "50") {
		t.Errorf("want a refusal naming the limit, got %q", got)
	}
}

// The vault a tool works on is one folder, and a path that leaves it is not a
// path this vault holds.
func TestAPathOutsideTheVaultIsRefused(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})

	got := failing(t, session, "note_rewrite", map[string]any{
		"path": "../../escaped.md", "body": "no\n", "fingerprint": "0-0",
	})
	if !strings.Contains(got, "vault") {
		t.Errorf("want a refusal about the vault, got %q", got)
	}
}

func TestTwoNotesOfOneNameAreReported(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"Entropy.md":         "# Entropy\n",
		"physics/Entropy.md": "# Entropy\n",
	})

	named := call[struct {
		Paths []string `json:"paths"`
	}](t, session, "note_resolve", map[string]any{"name": "Entropy"})
	if len(named.Paths) != 2 {
		t.Errorf("want both notes, got %v", named.Paths)
	}
}

// A dot in a name is part of the name, and the note is filed under all of it.
func TestANoteWhoseNameCarriesDotsIsFoundByIt(t *testing.T) {
	const lecture = "Seminar 1.2–1.3 — Lisbon, 9 July 1973"
	session, _ := connected(t, map[string]string{
		"notes/" + lecture + ".md": "# " + lecture + "\n",
	})

	named := call[struct {
		Paths []string `json:"paths"`
	}](t, session, "note_resolve", map[string]any{"name": lecture})
	if len(named.Paths) != 1 || named.Paths[0] != "notes/"+lecture+".md" {
		t.Errorf("got %v", named.Paths)
	}
}

// Whether a link is ambiguous is the resolver's answer and nobody else's: it
// depends on the whole vault, so it is asked at the moment it is wanted.
func TestALinkToASharedNameSaysItIsAmbiguous(t *testing.T) {
	// Neither is at the root, so neither is the exact path the name spells, and
	// neither sits beside the note that wrote the link. Only then is the name
	// left to answer for two notes at once.
	session, _ := connected(t, map[string]string{
		"physics/Entropy.md":   "# Entropy\n",
		"chemistry/Entropy.md": "# Entropy\n",
		"Heat.md":              "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
	})

	joined := call[struct {
		Links []mcp.Link `json:"links"`
	}](t, session, "link_list", map[string]any{"path": "Heat.md"})
	if len(joined.Links) != 1 {
		t.Fatalf("want the one link, got %+v", joined.Links)
	}
	if !joined.Links[0].Ambiguous {
		t.Errorf("two notes answer to this name: %+v", joined.Links[0])
	}
}

// A problem belongs to the note somebody opens to settle it, and says which
// check found it so a person can take one kind at a time.
func TestProblemsSayWhichCheckFoundThem(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"physics/Entropy.md":   "# Entropy\n",
		"chemistry/Entropy.md": "# Entropy\n",
		"Heat.md":              "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
		"Cold.md":              "---\nlinks:\n  - to: Nowhere\n    role: jump\n---\n# Cold\n",
	})

	found := call[struct {
		Problems []mcp.Problem `json:"problems"`
		Ran      []string      `json:"ran"`
	}](t, session, "vault_problems", nil)

	if len(found.Ran) == 0 {
		t.Error("an answer says which checks it covers")
	}
	for _, p := range found.Problems {
		if p.Check == "dangling" {
			t.Errorf("dangling is quiet unless asked for: %+v", p)
		}
	}
	ambiguous := 0
	for _, p := range found.Problems {
		if p.Check != "ambiguous" {
			continue
		}
		ambiguous++
		if p.Path != "Heat.md" || len(p.Candidates) != 2 {
			t.Errorf("want it against the note that wrote the link, with both targets: %+v", p)
		}
	}
	if ambiguous != 1 {
		t.Errorf("want the one ambiguous link, got %d", ambiguous)
	}

	quiet := call[struct {
		Problems []mcp.Problem `json:"problems"`
	}](t, session, "vault_problems", map[string]any{"checks": []string{"dangling"}})
	if len(quiet.Problems) != 1 || quiet.Problems[0].Path != "Cold.md" {
		t.Errorf("want the dangling link when asked for: %+v", quiet.Problems)
	}
}

// What note_read gives back is what note_rewrite takes: an agent that reads,
// edits and writes must not end up with the frontmatter inside the prose.
func TestReadingGivesBackOnlyTheProse(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"Entropy.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n# Entropy\n",
	})

	read := call[struct {
		Notes []mcp.Contents `json:"notes"`
	}](t, session, "note_read", map[string]any{"paths": []string{"Entropy.md"}})
	if strings.Contains(read.Notes[0].Body, "---") {
		t.Errorf("the frontmatter came back as prose:\n%s", read.Notes[0].Body)
	}

	// And a caller that hands back a whole note is told, rather than quietly
	// given a note with two frontmatter blocks in it.
	got := failing(t, session, "note_rewrite", map[string]any{
		"path":        "Entropy.md",
		"body":        "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n# Entropy\n\nMore.\n",
		"fingerprint": fingerprint(t, session, "Entropy.md"),
	})
	if !strings.Contains(got, "frontmatter") {
		t.Errorf("want a refusal naming the frontmatter, got %q", got)
	}
}

// read is one call of note_read, whole: what came back, what was not there and
// what was refused.
type reading struct {
	Notes   []mcp.Contents `json:"notes"`
	Missing []string       `json:"missing"`
	Refused []mcp.Refusal  `json:"refused"`
}

// put writes a file into a vault that is already being served, for the ones a
// scan would not index.
func put(t *testing.T, v domain.Vault, path, raw string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Path, filepath.FromSlash(path)), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The vault holds notes, and a folder holds whatever the person keeps in it. An
// agent asking for a book gets the refusal and not the book.
func TestReadingRefusesWhatIsNotANote(t *testing.T) {
	session, v := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})
	put(t, v, "library.epub", "PK\x03\x04 chapters of somebody else's book")

	got := call[reading](t, session, "note_read", map[string]any{
		"paths": []string{"Entropy.md", "library.epub"},
	})
	if len(got.Notes) != 1 || got.Notes[0].Path != "Entropy.md" {
		t.Fatalf("want the note and nothing else: %+v", got.Notes)
	}
	for _, c := range got.Notes {
		if strings.Contains(c.Body, "chapters") {
			t.Errorf("the bytes of a book were handed over:\n%s", c.Body)
		}
	}
	if len(got.Refused) != 1 || got.Refused[0].Path != "library.epub" {
		t.Fatalf("want the book refused by name: %+v", got.Refused)
	}
	if !strings.Contains(got.Refused[0].Why, "note") {
		t.Errorf("the refusal does not say what is wrong: %q", got.Refused[0].Why)
	}
}

// One byte that is not UTF-8 becomes U+FFFD wherever the answer is shown, and
// the next write puts those characters where the person's bytes were.
func TestReadingRefusesAFileThatIsNotText(t *testing.T) {
	session, v := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})
	put(t, v, "Pasted.md", "# Pasted\n\xff\xfe from somewhere\n")

	got := call[reading](t, session, "note_read", map[string]any{
		"paths": []string{"Entropy.md", "Pasted.md"},
	})
	if len(got.Notes) != 1 || got.Notes[0].Path != "Entropy.md" {
		t.Fatalf("want only the note that is text: %+v", got.Notes)
	}
	if len(got.Refused) != 1 || got.Refused[0].Path != "Pasted.md" {
		t.Fatalf("want the file refused by name: %+v", got.Refused)
	}
	if !strings.Contains(got.Refused[0].Why, "text") {
		t.Errorf("the refusal does not say what is wrong: %q", got.Refused[0].Why)
	}
}

// One note nobody can carry does not cost the others their answer.
func TestReadingSaysWhichNoteIsTooLargeAndCarriesOn(t *testing.T) {
	session, v := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})
	put(t, v, "Export.md", "# Export\n"+strings.Repeat("pasted in from somewhere ", note.MaxBytes/20))

	got := call[reading](t, session, "note_read", map[string]any{
		"paths": []string{"Entropy.md", "Export.md", "gone.md"},
	})
	if len(got.Notes) != 1 || got.Notes[0].Path != "Entropy.md" {
		t.Fatalf("the rest of the batch did not come back: %+v", got.Notes)
	}
	if len(got.Missing) != 1 || got.Missing[0] != "gone.md" {
		t.Errorf("want the path with no file behind it: %+v", got.Missing)
	}
	if len(got.Refused) != 1 || got.Refused[0].Path != "Export.md" {
		t.Fatalf("want the large note refused by name: %+v", got.Refused)
	}
	if !strings.Contains(got.Refused[0].Why, "open the file") {
		t.Errorf("the refusal does not say what to do instead: %q", got.Refused[0].Why)
	}
}

// A batch is many operations, not one. What happened to each has to come back,
// or a caller recovering from a partial failure starts by undoing what worked.
func TestABatchSaysWhatHappenedToEachNote(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"A.md":         "# A\n",
		"B.md":         "# B\n",
		"archive/B.md": "# B already here\n",
	})

	moved := call[struct {
		Moved []mcp.MoveOutcome `json:"moved"`
	}](t, session, "note_move", map[string]any{
		"paths": []string{"A.md", "B.md"}, "folder": "archive",
	})
	if len(moved.Moved) != 2 {
		t.Fatalf("want an outcome for each note, got %+v", moved.Moved)
	}
	if moved.Moved[0].Refused != "" || moved.Moved[0].To != "archive/A.md" {
		t.Errorf("the first one moved: %+v", moved.Moved[0])
	}
	if moved.Moved[1].Refused == "" {
		t.Errorf("the second one collided and should say so: %+v", moved.Moved[1])
	}
}
