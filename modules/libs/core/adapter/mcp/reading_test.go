package mcp_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// reads is every tool the reading server serves.
var reads = []string{
	"note_search", "note_get", "note_read", "note_neighbourhood",
	"link_list", "source_list", "source_read",
	"card_stencils", "card_read", "vault_get",
}

// The reading server stands on what it does not serve, so the list is exact:
// one writing tool reaching it fails this.
func TestTheReadingServerServesTheToolsThatRead(t *testing.T) {
	cfg, db := opened(t)
	v := testsupport.NewVault(t, kinetics())
	session := sessionOf(t, mcp.NewReading(reader(t, cfg, db, v)))
	exactly(t, serves(t, session), reads)
}

// An agent is told that everything it can call reads, and how to say where an
// answer came from, because the window it answers into opens no link.
func TestTheReadingInstructionsSayWhatThisAgentCanDo(t *testing.T) {
	cfg, db := opened(t)
	v := testsupport.NewVault(t, kinetics())
	session := sessionOf(t, mcp.NewReading(reader(t, cfg, db, v)))

	said := session.InitializeResult().Instructions
	for _, rule := range []string{"reads", "path", "heading", "chapter or page", "numen:"} {
		if !strings.Contains(said, rule) {
			t.Errorf("the instructions say nothing about %q:\n%s", rule, said)
		}
	}
}

// Every reading use case is here and no writing one is, so a tool that reached
// a writer would find nothing there.
func TestEveryReadingToolAnswersWithoutAWriter(t *testing.T) {
	cfg, db := opened(t)
	v := testsupport.NewVault(t, kinetics())
	session := sessionOf(t, mcp.NewReading(reader(t, cfg, db, v)))

	asked := []struct {
		name string
		args any
	}{
		{"note_search", map[string]any{"query": "microstates"}},
		{"note_get", map[string]any{"paths": []string{"Entropy.md"}}},
		{"note_read", map[string]any{"paths": []string{"Entropy.md"}}},
		{"note_neighbourhood", map[string]any{"path": "Entropy.md"}},
		{"link_list", map[string]any{"path": "Entropy.md"}},
		{"source_list", map[string]any{}},
		{"source_read", map[string]any{"path": "Entropy.md", "start": 0, "length": 200}},
		{"card_stencils", map[string]any{}},
		{"card_read", map[string]any{"path": "Kinetics.md"}},
		{"vault_get", map[string]any{}},
	}
	called := make([]string, 0, len(asked))
	for _, one := range asked {
		called = append(called, one.name)
		t.Run(one.name, func(t *testing.T) {
			call[json.RawMessage](t, session, one.name, one.args)
		})
	}
	exactly(t, called, serves(t, session))
}

// One installation holds every vault in one index, so a query that forgets its
// vault answers with another vault's notes and nothing fails. Two vaults whose
// prose shares no words, asked in both directions, is what says so.
func TestTheReadingToolsAnswerAboutOneVaultOnly(t *testing.T) {
	cfg, db := opened(t)
	physics := side{
		what:  "physics",
		vault: testsupport.NewVault(t, kinetics()),
		note:  "Entropy.md", deck: "Kinetics.md",
		called: "Randomness",
		own:    "microstates",
		hers:   []string{"microstates", "Randomness", "Thermodynamics", "engines"},
	}
	garden := side{
		what:  "garden",
		vault: testsupport.NewVault(t, beds()),
		note:  "Compost.md", deck: "Borders.md",
		called: "Heap",
		own:    "peelings",
		hers:   []string{"peelings", "Heap", "Mulch", "brambles"},
	}
	physics.session = sessionOf(t, mcp.NewReading(reader(t, cfg, db, physics.vault)))
	garden.session = sessionOf(t, mcp.NewReading(reader(t, cfg, db, garden.vault)))

	for _, pair := range []struct{ own, other side }{{physics, garden}, {garden, physics}} {
		t.Run(pair.own.what, func(t *testing.T) {
			pair.own.answers(t)
			pair.own.holdsNothingOf(t, pair.other)
		})
	}
}

// A side of the scoping test: one vault, the session serving it, the notes it
// is asked about, and the words that stand in it and in no other vault.
type side struct {
	what    string
	vault   domain.Vault
	session *sdk.ClientSession
	note    string
	deck    string
	// called is what that note is called, which is the whole of what note_get
	// carries beside the path it was asked by.
	called string
	// own is a word this vault's own answers must carry, so that a tool
	// answering nothing at all is not read as a vault kept apart.
	own string
	// hers is every word of this vault that no other vault's answer may hold.
	hers []string
}

// answers is this vault's own content, through every tool the reading server
// serves.
func (s side) answers(t *testing.T) {
	t.Helper()

	found := call[struct {
		Matches []mcp.Passage `json:"matches"`
	}](t, s.session, "note_search", map[string]any{"query": s.own})
	if len(found.Matches) == 0 {
		t.Errorf("%s does not find %q in its own vault", s.what, s.own)
	}

	looked := call[struct {
		Notes   []mcp.Note `json:"notes"`
		Missing []string   `json:"missing"`
	}](t, s.session, "note_get", map[string]any{"paths": []string{s.note}})
	if len(looked.Notes) != 1 || looked.Notes[0].Title != s.called {
		t.Errorf("%s does not look up its own note: %+v", s.what, looked)
	}

	got := call[reading](t, s.session, "note_read", map[string]any{"paths": []string{s.note}})
	if len(got.Notes) != 1 || !strings.Contains(got.Notes[0].Body, s.own) {
		t.Errorf("%s does not read its own note: %+v", s.what, got)
	}

	near := call[struct {
		Related []mcp.Neighbour `json:"related"`
	}](t, s.session, "note_neighbourhood", map[string]any{"path": s.note})
	if len(near.Related) == 0 {
		t.Errorf("%s sees nothing joined to its own note", s.what)
	}

	joined := call[struct {
		Links []mcp.Link `json:"links"`
	}](t, s.session, "link_list", map[string]any{"path": s.note})
	if len(joined.Links) == 0 {
		t.Errorf("%s sees no link written in its own note", s.what)
	}

	run := call[struct {
		Text string `json:"text"`
	}](t, s.session, "source_read", map[string]any{"path": s.note, "start": 0, "length": 400})
	if !strings.Contains(run.Text, s.own) {
		t.Errorf("%s reads its own note as a source without %q: %q", s.what, s.own, run.Text)
	}

	hand := dealt(t, s.session, map[string]any{"path": s.deck})
	if hand.Held == 0 {
		t.Errorf("%s reads no card out of its own deck", s.what)
	}
}

// holdsNothingOf asks this vault's session about the other vault's notes, and
// about the other vault's words, and reads every answer whole.
func (s side) holdsNothingOf(t *testing.T, other side) {
	t.Helper()

	for _, asked := range []struct {
		name string
		args any
	}{
		{"note_search", map[string]any{"query": other.own}},
		{"note_get", map[string]any{"paths": []string{other.note}}},
		{"note_read", map[string]any{"paths": []string{other.note}}},
		{"note_neighbourhood", map[string]any{"path": other.note}},
		{"link_list", map[string]any{"path": other.note}},
		{"source_read", map[string]any{"path": other.note, "start": 0, "length": 400}},
		{"card_read", map[string]any{"path": other.deck}},
	} {
		said := says(t, s.session, asked.name, asked.args)
		for _, word := range other.hers {
			if strings.Contains(said, word) {
				t.Errorf("%s answered %s with %q, which is %s's:\n%s",
					s.what, asked.name, word, other.what, said)
			}
		}
	}
}

// says is one call as it came back, whether it answered or refused. A vault
// leaks into any field of an answer, so the whole of it is read.
func says(t *testing.T, session *sdk.ClientSession, name string, args any) string {
	t.Helper()
	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return err.Error()
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// opened is one index, which is where every vault of an installation is held.
func opened(t *testing.T) (container.Config, *container.Index) {
	t.Helper()
	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return cfg, db
}

// reader scans a vault into the index and fills in what the reading tools work
// through. Every field that writes is left zero.
func reader(t *testing.T, cfg container.Config, db *container.Index, v domain.Vault) mcp.Core {
	t.Helper()

	readers := filesystem.VaultReaders{}
	scan := usecase.Scan{
		Readers: readers, Vaults: db.Vaults(), Notes: db.Notes(),
		Known: db.Queries(), Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	queries := db.Queries()
	cutting := cfg.Cards(queries, db.Links(), nil)

	return mcp.Core{
		Showing: mcp.One(v, v.Path),
		Readers: readers,
		Notes: mcp.Notes{
			Queries:       queries,
			Search:        search.New(db.Passages(), readers, nil, nil, nil, 0, nil),
			Neighbourhood: note.ShowNeighbourhood{Links: db.Links(), Notes: queries},
			Links:         note.ShowLinks{Links: db.Links()},
		},
		Cards:   mcp.Cards{Read: cutting.Read, List: cutting.List},
		Sources: mcp.Sources{Queries: db.SourcesKnown()},
	}
}

// kinetics and beds are two vaults whose prose shares no word, so a note of one
// showing up in the other's answer is unmistakable.
func kinetics() map[string]string {
	return map[string]string{
		"Entropy.md": "---\ntitle: Randomness\nlinks:\n  - to: Thermodynamics\n    role: parent\n---\n\n" +
			"Disorder counted across microstates.\n",
		"Thermodynamics.md": "# Thermodynamics\n\nHeat moving through engines.\n",
		"Physics.md": "---\ntype: stencil\nfields:\n  - Term\n  - Meaning\n---\n\n" +
			"## Recognise\n\n### Front\n\n{{Term}}\n\n### Back\n\n{{Meaning}}\n",
		"Kinetics.md": "---\ntype: deck\n---\n\n# Kinetics\n\n" +
			"## Entropy ^k7m2xq9fzp\n\n[[Physics]]\n\n### Term\n\nEntropy\n\n" +
			"### Meaning\n\nDisorder counted across microstates\n",
	}
}

func beds() map[string]string {
	return map[string]string{
		"Compost.md": "---\ntitle: Heap\nlinks:\n  - to: Mulch\n    role: parent\n---\n\n" +
			"Kitchen peelings rotting beneath straw.\n",
		"Mulch.md": "# Mulch\n\nBark laid over brambles.\n",
		"Garden.md": "---\ntype: stencil\nfields:\n  - Plant\n  - Habit\n---\n\n" +
			"## Recognise\n\n### Front\n\n{{Plant}}\n\n### Back\n\n{{Habit}}\n",
		"Borders.md": "---\ntype: deck\n---\n\n# Borders\n\n" +
			"## Compost ^3n8vr4tqch\n\n[[Garden]]\n\n### Plant\n\nCompost\n\n" +
			"### Habit\n\nKitchen peelings rotting beneath straw\n",
	}
}
