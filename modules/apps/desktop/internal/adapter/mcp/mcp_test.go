package mcp_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/mcp"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/lint"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// connected is the tools as an agent meets them: over a real session, through
// the protocol, rather than by calling the handlers directly.
func connected(t *testing.T, notes map[string]string) (*sdk.ClientSession, domain.Vault) {
	t.Helper()
	v, core := built(t, notes)

	server := mcp.New(core)
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
	return session, v
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
	db, err := container.Config{
		IndexPath: filepath.Join(t.TempDir(), "index.db"),
	}.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	readers, writers := filesystem.Readers{}, filesystem.Writers{}
	scan := usecase.Scan{
		Readers: readers, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.Queries(), Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	refresh := usecase.Refresh{Readers: readers, Notes: db.Notes()}
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	queries := db.Queries()

	core := mcp.Core{
		Vault: v, Root: v.Path, Readers: readers, Notes: queries,
		Search:        note.Search{Notes: queries},
		Neighbourhood: note.ShowNeighbourhood{Links: db.Links(), Notes: queries},
		Links:         note.ShowLinks{Links: db.Links()},
		Problems:      lint.Standard(db.Problems()),
		Create: note.Create{
			Writers: writers, Names: queries, Index: index,
		},
		Write:   note.Write{Readers: readers, Writers: writers, Index: index},
		Move:    note.Move{Readers: readers, Writers: writers, Links: db.Links(), Index: index},
		Remove:  note.Remove{Readers: readers, Writers: writers, Links: db.Links(), Index: index},
		Linking: note.Linking{Readers: readers, Writers: writers, Index: index},
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

func TestTheToolsAreNamedForWhatTheyWorkOn(t *testing.T) {
	session, _ := connected(t, nil)
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]bool{}
	for _, tool := range tools.Tools {
		named[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("%s has nothing to tell a model about when to use it", tool.Name)
		}
	}
	for _, want := range []string{
		"note_search", "note_get", "note_read", "note_neighbourhood",
		"note_create", "note_write", "note_rename", "note_move", "note_remove",
		"link_add", "link_update", "link_remove", "link_list",
		"vault_get", "vault_problems", "vault_named",
	} {
		if !named[want] {
			t.Errorf("%s is missing", want)
		}
	}
}

func TestANoteIsMadeAndFoundThroughTheTools(t *testing.T) {
	session, _ := connected(t, nil)

	created := call[note.Created](t, session, "note_create", map[string]any{
		"title": "Entropy", "body": "A measure of disorder.\n",
	})
	if created.Path != "Entropy.md" {
		t.Fatalf("want Entropy.md, got %s", created.Path)
	}

	found := call[struct {
		Matches []mcp.Note `json:"matches"`
	}](t, session, "note_search", map[string]any{"query": "disorder"})
	if len(found.Matches) != 1 || found.Matches[0].Path != created.Path {
		t.Errorf("a note made through the tools is not searchable: %+v", found.Matches)
	}
}

// A path that names nothing is an answer, not a failure: the note may have gone
// since the agent last looked.
func TestGetSeparatesWhatIsThereFromWhatIsNot(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})

	got := call[struct {
		Notes   []mcp.Note `json:"notes"`
		Missing []string   `json:"missing"`
	}](t, session, "note_get", map[string]any{
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

	if _, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "note_write",
		Arguments: map[string]any{
			"path": "Entropy.md", "body": "# Entropy\n\nRewritten.\n",
			"fingerprint": read.Notes[0].Fingerprint,
		},
	}); err != nil {
		t.Fatal(err)
	}

	again := call[struct {
		Notes []mcp.Contents `json:"notes"`
	}](t, session, "note_read", map[string]any{"paths": []string{"Entropy.md"}})
	if !strings.Contains(again.Notes[0].Body, "Rewritten.") {
		t.Errorf("the write did not land:\n%s", again.Notes[0].Body)
	}

	// The fingerprint is stale now, and the second write is refused.
	if got := failing(t, session, "note_write", map[string]any{
		"path": "Entropy.md", "body": "# Entropy\n\nAgain.\n",
		"fingerprint": read.Notes[0].Fingerprint,
	}); !strings.Contains(got, "changed") {
		t.Errorf("want a refusal naming the change, got %q", got)
	}
}

func TestLinkingTwoNotesShowsAtBothEnds(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n",
	})

	if _, err := session.CallTool(t.Context(), &sdk.CallToolParams{
		Name: "link_add",
		Arguments: map[string]any{
			"from": "Heat.md", "to": "Entropy", "role": "parent", "label": "follows from",
		},
	}); err != nil {
		t.Fatal(err)
	}

	around := call[struct {
		Focus   mcp.Note     `json:"focus"`
		Related []mcp.Seated `json:"related"`
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

func TestRemovingIsReversible(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})

	removed := call[struct {
		Removed []note.Removed `json:"removed"`
	}](t, session, "note_remove", map[string]any{"paths": []string{"Entropy.md"}})
	if len(removed.Removed) != 1 || removed.Removed[0].Trashed != ".trash/Entropy.md" {
		t.Fatalf("want the note in the trash: %+v", removed.Removed)
	}

	back := call[struct {
		Notes   []mcp.Note `json:"notes"`
		Missing []string   `json:"missing"`
	}](t, session, "note_get", map[string]any{"paths": []string{"Entropy.md"}})
	if len(back.Missing) != 1 {
		t.Errorf("a removed note is still in the index: %+v", back)
	}

}

// A ceiling that truncated in silence would read as "that is all there is".
func TestAskingForTooMuchIsRefusedRatherThanTrimmed(t *testing.T) {
	session, _ := connected(t, nil)

	paths := make([]string, 60)
	for i := range paths {
		paths[i] = "note.md"
	}
	if got := failing(t, session, "note_get", map[string]any{"paths": paths}); !strings.Contains(got, "50") {
		t.Errorf("want a refusal naming the limit, got %q", got)
	}
}

// The vault a tool works on is one folder, and a path that leaves it is not a
// path this vault holds.
func TestAPathOutsideTheVaultIsRefused(t *testing.T) {
	session, _ := connected(t, map[string]string{"Entropy.md": "# Entropy\n"})

	got := failing(t, session, "note_write", map[string]any{
		"path": "../../escaped.md", "body": "no\n",
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
	}](t, session, "vault_named", map[string]any{"name": "Entropy"})
	if len(named.Paths) != 2 {
		t.Errorf("want both notes, got %v", named.Paths)
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

// What note_read gives back is what note_write takes: an agent that reads,
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
	got := failing(t, session, "note_write", map[string]any{
		"path": "Entropy.md",
		"body": "---\nid: 01J8F3K2M9QRSTVWXYZ012\n---\n# Entropy\n\nMore.\n",
	})
	if !strings.Contains(got, "frontmatter") {
		t.Errorf("want a refusal naming the frontmatter, got %q", got)
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
