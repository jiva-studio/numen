package service_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// skyVault is the identity of the invented astronomy vault the tests of this file call.
const skyVault = "sky-vault"

type listedNote struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type noteListOutput struct {
	Notes      []listedNote `json:"notes"`
	Total      *int         `json:"total"`
	NextOffset int          `json:"next_offset"`
}

type noteTitlesOutput struct {
	Notes   []listedNote `json:"notes"`
	Missing []string     `json:"missing"`
}

type noteResolveOutput struct {
	Paths *[]string `json:"paths"`
}

type listedVault struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Folder  string `json:"folder"`
	Showing *bool  `json:"showing"`
}

type vaultListOutput struct {
	Vaults []listedVault `json:"vaults"`
}

type vaultProblem struct {
	Path   string `json:"path"`
	Check  string `json:"check"`
	Detail string `json:"detail"`
}

type vaultProblemsOutput struct {
	Problems []vaultProblem `json:"problems"`
	Ran      []string       `json:"ran"`
}

// newSkyVault is an invented vault of astronomy notes, with one file that is not a note and one
// note whose identifier does not parse.
func newSkyVault(t *testing.T) (*hosttest.Vault, *tools.Registry) {
	t.Helper()
	v := hosttest.New(skyVault)
	v.File("Sky/Orion.md", "---\ntitle: The Hunter Orion\n---\nOrion rises in winter.")
	v.File("Sky/Comets.md", "---\ntitle: Comets and Their Tails\n---\nA tail points away from the sun.")
	v.File("Sky/Deep/Nebula.md", "---\ntitle: The Crab Nebula\n---\nRemnant of a supernova.")
	v.File("Skyline.md", "---\ntitle: City Skyline at Night\n---\nLight hides the stars.")
	v.File("Tides.md", "---\ntitle: Tides and the Moon\n---\nThe moon pulls the sea.")
	v.File("Sky/chart.txt", "RA 05h 35m\n")
	registry, err := service.New(v).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return v, registry
}

// callTool calls the tool on the vault and returns its structured result, failing the test when the
// tool is unknown or refuses.
func callTool(t *testing.T, registry *tools.Registry, vaultID, name, arguments string) []byte {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.GetError() != tools.ErrorCodeUnspecified {
		t.Fatalf("call %s %s: error code %v", name, arguments, res.GetError())
	}
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("call %s %s refused: %s", name, arguments, res.GetResult().GetText())
	}
	return res.GetResult().GetStructured()
}

// requireRefusal calls the tool and fails the test unless the tool is known and refuses the call.
func requireRefusal(t *testing.T, registry *tools.Registry, vaultID, name, arguments string) {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.GetError() == tools.ErrorCodeUnknownTool {
		t.Fatalf("call %s: the tool is unknown", name)
	}
	if !res.GetResult().GetIsRefusal() && res.GetError() == tools.ErrorCodeUnspecified {
		t.Errorf("call %s %s was not refused: %s", name, arguments, res.GetResult().GetStructured())
	}
}

func decodeInto(t *testing.T, structured []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(structured, out); err != nil {
		t.Fatalf("unmarshal %s: %v", structured, err)
	}
}

func getPaths(notes []listedNote) []string {
	out := make([]string, 0, len(notes))
	for _, n := range notes {
		out = append(out, n.Path)
	}
	return out
}

func TestNoteListReturnsNotesOrderedByPathWithTitles(t *testing.T) {
	_, registry := newSkyVault(t)
	var out noteListOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_list", `{}`), &out)

	want := []listedNote{
		{Path: "Sky/Comets.md", Title: "Comets and Their Tails"},
		{Path: "Sky/Deep/Nebula.md", Title: "The Crab Nebula"},
		{Path: "Sky/Orion.md", Title: "The Hunter Orion"},
		{Path: "Skyline.md", Title: "City Skyline at Night"},
		{Path: "Tides.md", Title: "Tides and the Moon"},
	}
	if !slices.Equal(out.Notes, want) {
		t.Errorf("notes = %v, want %v", out.Notes, want)
	}
	if out.Total == nil || *out.Total != 5 {
		t.Errorf("total = %v, want 5", out.Total)
	}
	if out.NextOffset != 0 {
		t.Errorf("next_offset = %d, want 0 when nothing is left", out.NextOffset)
	}
}

func TestNoteListFiltersByFolderAtAnyDepth(t *testing.T) {
	_, registry := newSkyVault(t)
	var out noteListOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_list", `{"folder":"Sky"}`), &out)

	want := []string{"Sky/Comets.md", "Sky/Deep/Nebula.md", "Sky/Orion.md"}
	if got := getPaths(out.Notes); !slices.Equal(got, want) {
		t.Errorf("paths under Sky = %v, want %v", got, want)
	}
	if out.Total == nil || *out.Total != 3 {
		t.Errorf("total = %v, want 3", out.Total)
	}
}

func TestNoteListPagesWithLimitAndOffset(t *testing.T) {
	_, registry := newSkyVault(t)

	var first noteListOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_list", `{"limit":2}`), &first)
	if got, want := getPaths(first.Notes), []string{"Sky/Comets.md", "Sky/Deep/Nebula.md"}; !slices.Equal(got, want) {
		t.Errorf("first page = %v, want %v", got, want)
	}
	if first.Total == nil || *first.Total != 5 {
		t.Errorf("total = %v, want 5 (the count before limit)", first.Total)
	}
	if first.NextOffset != 2 {
		t.Errorf("next_offset = %d, want 2", first.NextOffset)
	}

	var last noteListOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_list", `{"limit":2,"offset":4}`), &last)
	if got, want := getPaths(last.Notes), []string{"Tides.md"}; !slices.Equal(got, want) {
		t.Errorf("last page = %v, want %v", got, want)
	}
	if last.Total == nil || *last.Total != 5 {
		t.Errorf("total = %v, want 5 (the count before offset)", last.Total)
	}
	if last.NextOffset != 0 {
		t.Errorf("next_offset = %d, want 0 when nothing is left", last.NextOffset)
	}
}

func TestNoteListReturnsAtMostOneHundredNotesByDefault(t *testing.T) {
	v := hosttest.New(skyVault)
	for i := range 120 {
		v.File(fmt.Sprintf("Stars/Star %03d.md", i), fmt.Sprintf("# Star %03d\nA star.", i))
	}
	registry, err := service.New(v).Registry()
	if err != nil {
		t.Fatal(err)
	}
	var out noteListOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_list", `{}`), &out)
	if len(out.Notes) != 100 {
		t.Errorf("got %d notes, want 100", len(out.Notes))
	}
	if out.Total == nil || *out.Total != 120 {
		t.Errorf("total = %v, want 120", out.Total)
	}
	if out.NextOffset != 100 {
		t.Errorf("next_offset = %d, want 100", out.NextOffset)
	}
}

func TestNoteListRefusesALimitAboveFiveHundred(t *testing.T) {
	_, registry := newSkyVault(t)
	requireRefusal(t, registry, skyVault, "note_list", `{"limit":501}`)
	callTool(t, registry, skyVault, "note_list", `{"limit":500}`)
}

func TestNoteTitlesReturnsTitlesAndMissingPaths(t *testing.T) {
	_, registry := newSkyVault(t)
	var out noteTitlesOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_titles",
		`{"paths":["Tides.md","Sky/Orion.md","Sky/Pulsar.md"]}`), &out)

	got := map[string]string{}
	for _, n := range out.Notes {
		got[n.Path] = n.Title
	}
	want := map[string]string{"Tides.md": "Tides and the Moon", "Sky/Orion.md": "The Hunter Orion"}
	if len(got) != len(want) || got["Tides.md"] != want["Tides.md"] || got["Sky/Orion.md"] != want["Sky/Orion.md"] {
		t.Errorf("titles = %v, want %v", got, want)
	}
	if !slices.Equal(out.Missing, []string{"Sky/Pulsar.md"}) {
		t.Errorf("missing = %v, want [Sky/Pulsar.md]", out.Missing)
	}
}

func TestNoteTitlesRefusesMoreThanFiftyPaths(t *testing.T) {
	_, registry := newSkyVault(t)
	paths := make([]string, 51)
	for i := range paths {
		paths[i] = fmt.Sprintf("Sky/Star %d.md", i)
	}
	arguments, err := json.Marshal(map[string][]string{"paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	requireRefusal(t, registry, skyVault, "note_titles", string(arguments))

	fifty, err := json.Marshal(map[string][]string{"paths": paths[:50]})
	if err != nil {
		t.Fatal(err)
	}
	var out noteTitlesOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_titles", string(fifty)), &out)
	if len(out.Missing) != 50 {
		t.Errorf("got %d missing paths, want 50", len(out.Missing))
	}
}

func TestNoteResolveReturnsEveryNoteWithTheName(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("A/Comet.md", "# Comet\nHalley.")
	v.File("B/Comet.md", "# Comet\nHale-Bopp.")

	var out noteResolveOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_resolve", `{"name":"Comet"}`), &out)
	if out.Paths == nil {
		t.Fatal("the result has no paths")
	}
	got := slices.Clone(*out.Paths)
	slices.Sort(got)
	if want := []string{"A/Comet.md", "B/Comet.md"}; !slices.Equal(got, want) {
		t.Errorf("paths = %v, want %v", got, want)
	}
}

func TestNoteResolveReturnsEmptyListForUnknownName(t *testing.T) {
	_, registry := newSkyVault(t)
	var out noteResolveOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_resolve", `{"name":"Quasar"}`), &out)
	if out.Paths == nil {
		t.Fatal("the result has no paths list")
	}
	if len(*out.Paths) != 0 {
		t.Errorf("paths = %v, want none", *out.Paths)
	}
}

func TestVaultListReturnsEveryVaultAndMarksTheCalledOne(t *testing.T) {
	sky, registry := newSkyVault(t)
	garden := hosttest.New("garden-vault")
	garden.File("Beds.md", "# Beds\nRaised beds of oak.")
	sky.Beside(garden)

	var out vaultListOutput
	decodeInto(t, callTool(t, registry, "garden-vault", "vault_list", `{}`), &out)
	if len(out.Vaults) != 2 {
		t.Fatalf("got %d vaults, want 2: %v", len(out.Vaults), out.Vaults)
	}
	byID := map[string]listedVault{}
	for _, one := range out.Vaults {
		byID[one.ID] = one
	}
	for id, want := range map[string]struct {
		folder  string
		showing bool
	}{
		"garden-vault": {folder: "/vaults/garden-vault", showing: true},
		skyVault:       {folder: "/vaults/" + skyVault, showing: false},
	} {
		got, ok := byID[id]
		if !ok {
			t.Errorf("vault %s is not listed", id)
			continue
		}
		if got.Name != id {
			t.Errorf("vault %s name = %q, want %q", id, got.Name, id)
		}
		if got.Folder != want.folder {
			t.Errorf("vault %s folder = %q, want %q", id, got.Folder, want.folder)
		}
		if got.Showing == nil || *got.Showing != want.showing {
			t.Errorf("vault %s showing = %v, want %v", id, got.Showing, want.showing)
		}
	}
}

func TestVaultProblemsRunsTheParseCheckByDefault(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("Sky/Pulsar.md", "---\nid: not-an-identifier\n---\n# Pulsar\nIt spins.")

	var out vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problems", `{}`), &out)
	if !slices.Equal(out.Ran, []string{"parse"}) {
		t.Errorf("ran = %v, want [parse]", out.Ran)
	}
	if len(out.Problems) != 1 {
		t.Fatalf("got %d problems, want 1: %v", len(out.Problems), out.Problems)
	}
	got := out.Problems[0]
	if got.Path != "Sky/Pulsar.md" || got.Check != "parse" || !strings.Contains(got.Detail, "not-an-identifier") {
		t.Errorf("problem = %+v, want Sky/Pulsar.md, parse, a detail naming not-an-identifier", got)
	}
}

func TestVaultProblemsRunsTheParseCheckWhenNamed(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("Sky/Pulsar.md", "---\nid: not-an-identifier\n---\n# Pulsar\nIt spins.")

	var out vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problems", `{"checks":["parse"]}`), &out)
	if !slices.Equal(out.Ran, []string{"parse"}) {
		t.Errorf("ran = %v, want [parse]", out.Ran)
	}
	if len(out.Problems) != 1 || out.Problems[0].Path != "Sky/Pulsar.md" {
		t.Errorf("problems = %v, want one for Sky/Pulsar.md", out.Problems)
	}
}

func TestVaultProblemsReturnsNoProblemsForValidNotes(t *testing.T) {
	_, registry := newSkyVault(t)
	var out vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problems", `{}`), &out)
	if len(out.Problems) != 0 {
		t.Errorf("problems = %v, want none", out.Problems)
	}
	if !slices.Equal(out.Ran, []string{"parse"}) {
		t.Errorf("ran = %v, want [parse]", out.Ran)
	}
}

func TestVaultProblemsRefusesACheckOtherThanParse(t *testing.T) {
	_, registry := newSkyVault(t)
	requireRefusal(t, registry, skyVault, "vault_problems", `{"checks":["dangling"]}`)
	requireRefusal(t, registry, skyVault, "vault_problems", `{"checks":["parse","dangling"]}`)
}

// getArrangeToolNames returns the note_list group: the tools that list, title and resolve notes, list vaults and check problems.
func getArrangeToolNames() []string {
	return []string{"note_list", "note_titles", "note_resolve", "vault_list", "vault_problems"}
}

func TestNoteListGroupIsOnReadingAndFullServers(t *testing.T) {
	_, registry := newSkyVault(t)
	infos, err := registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	servers := map[string][]tools.Server{}
	for _, info := range infos {
		servers[info.GetName()] = info.GetServers()
	}
	for _, name := range getArrangeToolNames() {
		got, ok := servers[name]
		if !ok {
			t.Errorf("the tool %s is not listed", name)
			continue
		}
		if !slices.Contains(got, tools.Reading) {
			t.Errorf("the tool %s is not on the Reading server: %v", name, got)
		}
		if !slices.Contains(got, tools.Full) {
			t.Errorf("the tool %s is not on the Full server: %v", name, got)
		}
	}
}

func TestNoteListGroupRefusesCallWithoutVault(t *testing.T) {
	_, registry := newSkyVault(t)
	arguments := map[string]string{
		"note_list":      `{}`,
		"note_titles":    `{"paths":["Tides.md"]}`,
		"note_resolve":   `{"name":"Tides"}`,
		"vault_list":     `{}`,
		"vault_problems": `{}`,
	}
	for _, name := range getArrangeToolNames() {
		res, err := registry.InvokeTool(t.Context(), tools.Call{Name: name, Arguments: []byte(arguments[name])})
		if err != nil {
			t.Fatalf("call %s: %v", name, err)
		}
		if res.GetError() == tools.ErrorCodeUnknownTool {
			t.Errorf("call %s: the tool is unknown", name)
			continue
		}
		if !res.GetResult().GetIsRefusal() || res.GetResult().GetText() != "the call names no vault" {
			t.Errorf("a call of %s naming no vault answered %v", name, res.GetResult())
		}
	}
}
