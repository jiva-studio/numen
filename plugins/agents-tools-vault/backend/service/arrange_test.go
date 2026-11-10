package service_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// skyVault is the identity of the invented astronomy vault the tests of this file call.
const skyVault = "sky-vault"

type noteEntry struct {
	Path  string `json:"path"`
	Title string `json:"title"`
}

type noteListOutput struct {
	Notes      []noteEntry `json:"notes"`
	Total      *int        `json:"total"`
	NextOffset int         `json:"nextOffset"`
}

type noteTitlesOutput struct {
	Notes        []noteEntry `json:"notes"`
	MissingPaths []string    `json:"missingPaths"`
}

type noteTitle struct {
	Path  string  `json:"path"`
	Title string  `json:"title"`
	ID    *string `json:"id"`
}

type noteTitlesWithIDsOutput struct {
	Notes []noteTitle `json:"notes"`
}

type noteResolveOutput struct {
	Paths *[]string `json:"paths"`
}

type vaultEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsCurrent *bool  `json:"isCurrent"`
	IsMissing *bool  `json:"isMissing"`
}

type vaultListOutput struct {
	Vaults []vaultEntry `json:"vaults"`
}

type vaultProblem struct {
	Path       string   `json:"path"`
	Check      string   `json:"check"`
	Detail     string   `json:"detail"`
	Address    string   `json:"address"`
	Candidates []string `json:"candidates"`
}

type vaultProblemsOutput struct {
	Problems []vaultProblem `json:"problems"`
	Checks   []string       `json:"checks"`
}

func newSkyVault(t *testing.T) (*hosttest.Vault, *tools.Registry) {
	t.Helper()
	v := hosttest.New(skyVault)
	v.File("Sky/Orion.md", "---\ntitle: The Hunter Orion\n---\nOrion rises in winter.")
	v.File("Sky/Comets.md", "---\ntitle: Comets and Their Tails\n---\nA tail points away from the sun.")
	v.File("Sky/Deep/Nebula.md", "---\ntitle: The Crab Nebula\n---\nRemnant of a supernova.")
	v.File("Skyline.md", "---\ntitle: City Skyline at Night\n---\nLight hides the stars.")
	v.File("Tides.md", "---\ntitle: Tides and the Moon\n---\nThe moon pulls the sea.")
	v.File("Sky/chart.txt", "RA 05h 35m\n")
	registry, err := service.New(vaultsOf(v)).WithVaultList(v).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return v, registry
}

func callTool(t *testing.T, registry *tools.Registry, vaultID, name, arguments string) []byte {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.GetError() != tools.ErrorCodeUnspecified {
		t.Fatalf("call %s %s: error code %v", name, arguments, res.GetError())
	}
	if res.GetResult().GetIsError() {
		t.Fatalf("call %s %s rejected: %s", name, arguments, res.GetResult().GetText())
	}
	return res.GetResult().GetStructured()
}

func requireRejection(t *testing.T, registry *tools.Registry, vaultID, name, arguments string) {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: vaultID, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.GetError() == tools.ErrorCodeUnknownTool {
		t.Fatalf("call %s: the tool is unknown", name)
	}
	if !res.GetResult().GetIsError() && res.GetError() == tools.ErrorCodeUnspecified {
		t.Errorf("call %s %s was not rejected: %s", name, arguments, res.GetResult().GetStructured())
	}
}

func decodeInto(t *testing.T, structured []byte, out any) {
	t.Helper()
	if err := json.Unmarshal(structured, out); err != nil {
		t.Fatalf("unmarshal %s: %v", structured, err)
	}
}

func getPaths(notes []noteEntry) []string {
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

	want := []noteEntry{
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
		t.Errorf("nextOffset = %d, want 0 when nothing is left", out.NextOffset)
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
		t.Errorf("nextOffset = %d, want 2", first.NextOffset)
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
		t.Errorf("nextOffset = %d, want 0 when nothing is left", last.NextOffset)
	}
}

func TestNoteListReturnsAtMostOneHundredNotesByDefault(t *testing.T) {
	v := hosttest.New(skyVault)
	for i := range 120 {
		v.File(fmt.Sprintf("Stars/Star %03d.md", i), fmt.Sprintf("# Star %03d\nA star.", i))
	}
	registry, err := service.New(vaultsOf(v)).Registry()
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
		t.Errorf("nextOffset = %d, want 100", out.NextOffset)
	}
}

func TestNoteListRejectsALimitAboveFiveHundred(t *testing.T) {
	_, registry := newSkyVault(t)
	requireRejection(t, registry, skyVault, "note_list", `{"limit":501}`)
	callTool(t, registry, skyVault, "note_list", `{"limit":500}`)
}

func TestNoteTitleListReturnsTitlesAndMissingPaths(t *testing.T) {
	_, registry := newSkyVault(t)
	var out noteTitlesOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_title_list",
		`{"paths":["Tides.md","Sky/Orion.md","Sky/Pulsar.md"]}`), &out)

	got := map[string]string{}
	for _, n := range out.Notes {
		got[n.Path] = n.Title
	}
	want := map[string]string{"Tides.md": "Tides and the Moon", "Sky/Orion.md": "The Hunter Orion"}
	if len(got) != len(want) || got["Tides.md"] != want["Tides.md"] || got["Sky/Orion.md"] != want["Sky/Orion.md"] {
		t.Errorf("titles = %v, want %v", got, want)
	}
	if !slices.Equal(out.MissingPaths, []string{"Sky/Pulsar.md"}) {
		t.Errorf("missingPaths = %v, want [Sky/Pulsar.md]", out.MissingPaths)
	}
}

func TestNoteTitleListReturnsTheIDOfANoteThatHasOne(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("Sky/Pulsar.md", "---\nid: 01H7X1234567890ABCDEFGHJKM\ntitle: The Vela Pulsar\n---\nIt spins.")

	var out noteTitlesWithIDsOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_title_list", `{"paths":["Sky/Pulsar.md","Tides.md"]}`), &out)
	byPath := map[string]noteTitle{}
	for _, n := range out.Notes {
		byPath[n.Path] = n
	}
	pulsar, ok := byPath["Sky/Pulsar.md"]
	if !ok {
		t.Fatalf("Sky/Pulsar.md is not in %+v", out.Notes)
	}
	if pulsar.ID == nil || *pulsar.ID != "01H7X1234567890ABCDEFGHJKM" {
		t.Errorf("id of Sky/Pulsar.md = %v, want 01H7X1234567890ABCDEFGHJKM", pulsar.ID)
	}
	tides, ok := byPath["Tides.md"]
	if !ok {
		t.Fatalf("Tides.md is not in %+v", out.Notes)
	}
	if tides.ID != nil {
		t.Errorf("id of Tides.md = %q, want it left out for a note without an identifier", *tides.ID)
	}
}

func TestNoteTitleListRejectsMoreThanFiftyPaths(t *testing.T) {
	_, registry := newSkyVault(t)
	paths := make([]string, 51)
	for i := range paths {
		paths[i] = fmt.Sprintf("Sky/Star %d.md", i)
	}
	arguments, err := json.Marshal(map[string][]string{"paths": paths})
	if err != nil {
		t.Fatal(err)
	}
	requireRejection(t, registry, skyVault, "note_title_list", string(arguments))

	fifty, err := json.Marshal(map[string][]string{"paths": paths[:50]})
	if err != nil {
		t.Fatal(err)
	}
	var out noteTitlesOutput
	decodeInto(t, callTool(t, registry, skyVault, "note_title_list", string(fifty)), &out)
	if len(out.MissingPaths) != 50 {
		t.Errorf("got %d missing paths, want 50", len(out.MissingPaths))
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
	byID := map[string]vaultEntry{}
	for _, one := range out.Vaults {
		byID[one.ID] = one
	}
	for id, want := range map[string]struct {
		path      string
		isCurrent bool
	}{
		"garden-vault": {path: "/vaults/garden-vault", isCurrent: true},
		skyVault:       {path: "/vaults/" + skyVault, isCurrent: false},
	} {
		got, ok := byID[id]
		if !ok {
			t.Errorf("vault %s is not listed", id)
			continue
		}
		if got.Name != id {
			t.Errorf("vault %s name = %q, want %q", id, got.Name, id)
		}
		if got.Path != want.path {
			t.Errorf("vault %s path = %q, want %q", id, got.Path, want.path)
		}
		if got.IsCurrent == nil || *got.IsCurrent != want.isCurrent {
			t.Errorf("vault %s isCurrent = %v, want %v", id, got.IsCurrent, want.isCurrent)
		}
	}
}

func TestVaultListMarksAVaultWithoutItsFolderAsMissing(t *testing.T) {
	sky, registry := newSkyVault(t)
	garden := hosttest.New("garden-vault")
	sky.Beside(garden)
	garden.SetMissing(true)

	var out vaultListOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_list", `{}`), &out)
	byID := map[string]vaultEntry{}
	for _, one := range out.Vaults {
		byID[one.ID] = one
	}
	if got, ok := byID["garden-vault"]; !ok || got.IsMissing == nil || !*got.IsMissing {
		t.Errorf("garden-vault = %+v, want it listed with isMissing true", got)
	}
	if got, ok := byID[skyVault]; !ok || (got.IsMissing != nil && *got.IsMissing) {
		t.Errorf("%s = %+v, want it listed and not missing", skyVault, got)
	}
}

func getDefaultChecks() []string {
	return []string{"parse", "frontmatter", "ambiguous"}
}

func filterProblems(problems []vaultProblem, check string) []vaultProblem {
	var out []vaultProblem
	for _, p := range problems {
		if p.Check == check {
			out = append(out, p)
		}
	}
	return out
}

func TestVaultProblemListRunsThreeChecksByDefault(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("Sky/Pulsar.md", "---\nid: not-an-identifier\n---\n# Pulsar\nIt spins.")

	var out vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problem_list", `{}`), &out)
	if !slices.Equal(out.Checks, getDefaultChecks()) {
		t.Errorf("checks = %v, want %v", out.Checks, getDefaultChecks())
	}
	parse := filterProblems(out.Problems, "parse")
	if len(parse) != 1 {
		t.Fatalf("got %d parse problems, want 1: %v", len(parse), out.Problems)
	}
	if got := parse[0]; got.Path != "Sky/Pulsar.md" || !strings.Contains(got.Detail, "not-an-identifier") {
		t.Errorf("problem = %+v, want Sky/Pulsar.md, a detail naming not-an-identifier", got)
	}
}

func TestVaultProblemListRunsOnlyTheNamedParseCheck(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("Sky/Pulsar.md", "---\nid: not-an-identifier\n---\n# Pulsar\nIt spins.")

	var out vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problem_list", `{"checks":["parse"]}`), &out)
	if !slices.Equal(out.Checks, []string{"parse"}) {
		t.Errorf("checks = %v, want [parse]", out.Checks)
	}
	if len(out.Problems) != 1 || out.Problems[0].Path != "Sky/Pulsar.md" || out.Problems[0].Check != "parse" {
		t.Errorf("problems = %v, want one parse problem for Sky/Pulsar.md", out.Problems)
	}
}

func TestVaultProblemListReturnsNoProblemsForValidNotes(t *testing.T) {
	_, registry := newSkyVault(t)
	var out vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problem_list", `{}`), &out)
	if len(out.Problems) != 0 {
		t.Errorf("problems = %v, want none", out.Problems)
	}
	if !slices.Equal(out.Checks, getDefaultChecks()) {
		t.Errorf("checks = %v, want %v", out.Checks, getDefaultChecks())
	}
}

func TestVaultProblemListReturnsAmbiguousLinkCandidates(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("A/Comet.md", "# Comet\nHalley.")
	v.File("B/Comet.md", "# Comet\nHale-Bopp.")
	v.File("Tides.md", "---\ntitle: Tides and the Moon\n---\nThe moon pulls the sea, unlike a [[Comet]].")

	var out vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problem_list", `{}`), &out)
	ambiguous := filterProblems(out.Problems, "ambiguous")
	if len(ambiguous) != 1 {
		t.Fatalf("got %d ambiguous problems, want 1: %v", len(ambiguous), out.Problems)
	}
	got := ambiguous[0]
	if got.Path != "Tides.md" {
		t.Errorf("path = %q, want Tides.md", got.Path)
	}
	if !strings.Contains(got.Address, "Comet") {
		t.Errorf("address = %q, want one naming Comet", got.Address)
	}
	candidates := slices.Clone(got.Candidates)
	slices.Sort(candidates)
	if want := []string{"A/Comet.md", "B/Comet.md"}; !slices.Equal(candidates, want) {
		t.Errorf("candidates = %v, want %v", got.Candidates, want)
	}
}

func TestVaultProblemListRunsTheDanglingCheckOnlyWhenNamed(t *testing.T) {
	v, registry := newSkyVault(t)
	v.File("Tides.md", "---\ntitle: Tides and the Moon\n---\nThe moon pulls the sea, see [[Quasar]].")

	var byDefault vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problem_list", `{}`), &byDefault)
	if slices.Contains(byDefault.Checks, "dangling") || len(filterProblems(byDefault.Problems, "dangling")) != 0 {
		t.Errorf("the default run checked dangling links: %+v", byDefault)
	}

	var named vaultProblemsOutput
	decodeInto(t, callTool(t, registry, skyVault, "vault_problem_list", `{"checks":["dangling"]}`), &named)
	if !slices.Equal(named.Checks, []string{"dangling"}) {
		t.Errorf("checks = %v, want [dangling]", named.Checks)
	}
	if len(named.Problems) != 1 {
		t.Fatalf("got %d problems, want 1: %v", len(named.Problems), named.Problems)
	}
	if got := named.Problems[0]; got.Path != "Tides.md" || got.Check != "dangling" || !strings.Contains(got.Address, "Quasar") {
		t.Errorf("problem = %+v, want Tides.md, dangling, an address naming Quasar", got)
	}
}

func TestVaultProblemListOfAnUnknownCheckNameReturnsTheHostError(t *testing.T) {
	_, registry := newSkyVault(t)
	want := `"spelling" is not a check; the checks are parse, frontmatter, ambiguous, dangling`
	for _, arguments := range []string{`{"checks":["spelling"]}`, `{"checks":["parse","spelling"]}`} {
		if text := getRejectionText(t, registry, skyVault, "vault_problem_list", arguments); !strings.Contains(text, want) {
			t.Errorf("vault_problem_list %s gave %q, want the host's error %q", arguments, text, want)
		}
	}
}

func getArrangeToolNames() []string {
	return []string{"note_list", "note_title_list", "note_resolve", "vault_list", "vault_problem_list"}
}

func TestNoteListGroupIsOnReadingAndFullServers(t *testing.T) {
	_, registry := newSkyVault(t)
	infos, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	servers := map[string][]tools.Server{}
	for _, info := range infos.Tools {
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

func TestNoteListGroupRejectsCallWithoutVault(t *testing.T) {
	_, registry := newSkyVault(t)
	arguments := map[string]string{
		"note_list":          `{}`,
		"note_title_list":    `{"paths":["Tides.md"]}`,
		"note_resolve":       `{"name":"Tides"}`,
		"vault_list":         `{}`,
		"vault_problem_list": `{}`,
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
		if !res.GetResult().GetIsError() || res.GetResult().GetText() != name+": the call names no vault" {
			t.Errorf("a call of %s naming no vault returned %v", name, res.GetResult())
		}
	}
}
