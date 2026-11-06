package service_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

type linkAddItem struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Fingerprint string `json:"fingerprint"`
	Error       string `json:"error"`
}

type linkAddOutput struct {
	Added []linkAddItem `json:"added"`
}

type linkEditOutput struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// readFingerprint is the fingerprint note_read gives for the note.
func readFingerprint(t *testing.T, registry *tools.Registry, notePath string) string {
	t.Helper()
	var out struct {
		Fingerprint string `json:"fingerprint"`
	}
	decodeInto(t, callTool(t, registry, geologyVault, "note_read", fmt.Sprintf(`{"path":%q}`, notePath)), &out)
	if out.Fingerprint == "" {
		t.Fatalf("note_read gave no fingerprint for %s", notePath)
	}
	return out.Fingerprint
}

// readLinks is the links written in the frontmatter of the note as the vault holds it now.
func readLinks(t *testing.T, v *hosttest.Vault, notePath string) []domain.Link {
	t.Helper()
	content, ok := v.Content(notePath)
	if !ok {
		t.Fatalf("%s is not in the vault", notePath)
	}
	return markdown.Parse(domain.FileStat{Path: notePath}, []byte(content)).Links
}

// findLink is the link of the note to the name with the role, and false when there is none.
func findLink(links []domain.Link, to string, role domain.LinkRole) (domain.Link, bool) {
	for _, l := range links {
		if l.Target.Value == to && l.Role == role {
			return l, true
		}
	}
	return domain.Link{}, false
}

// newLinkedGeologyVault is the geology vault whose Basalt note links to Granite twice, as a jump
// with a label and a note and as a reference, and to Minerals as its parent.
func newLinkedGeologyVault(t *testing.T) (*hosttest.Vault, *tools.Registry) {
	t.Helper()
	v, registry := newGeologyVault(t)
	v.File("Rocks/Basalt.md", "---\ntitle: Basalt\nlinks:\n"+
		"  - to: Granite\n    role: jump\n    label: cousin\n    note: both are igneous\n"+
		"  - to: Granite\n    role: ref\n"+
		"  - to: Minerals\n    role: parent\n"+
		"---\nA volcanic rock.\n")
	if links := readLinks(t, v, "Rocks/Basalt.md"); len(links) != 3 {
		t.Fatalf("the fixture has %d links, want 3: %+v", len(links), links)
	}
	return v, registry
}

func TestLinkAddWritesTheLinksIntoTheFrontmatterOfEachNote(t *testing.T) {
	v, registry := newGeologyVault(t)
	basalt := readFingerprint(t, registry, "Rocks/Basalt.md")
	minerals := readFingerprint(t, registry, "Minerals.md")

	arguments := fmt.Sprintf(`{"links":[
		{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Granite","role":"jump","label":"cousin","note":"both are igneous"},
		{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Minerals","role":"parent","type":"source"},
		{"from":"Minerals.md","fingerprint":%q,"to":"Granite","role":"child"}]}`, basalt, basalt, minerals)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
	if len(out.Added) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(out.Added), out.Added)
	}
	for i, want := range []struct{ from, to string }{
		{"Rocks/Basalt.md", "Granite"}, {"Rocks/Basalt.md", "Minerals"}, {"Minerals.md", "Granite"},
	} {
		if got := out.Added[i]; got.From != want.from || got.To != want.to || got.Error != "" || got.Fingerprint == "" {
			t.Errorf("result %d = %+v, want %s to %s with a fingerprint and no error", i, got, want.from, want.to)
		}
	}
	if out.Added[0].Fingerprint != out.Added[1].Fingerprint {
		t.Errorf("the links into Rocks/Basalt.md carry fingerprints %q and %q, want one write",
			out.Added[0].Fingerprint, out.Added[1].Fingerprint)
	}
	if now := readFingerprint(t, registry, "Rocks/Basalt.md"); now != out.Added[0].Fingerprint {
		t.Errorf("fingerprint of Rocks/Basalt.md = %q, want the one returned, %q", now, out.Added[0].Fingerprint)
	}

	content, _ := v.Content("Rocks/Basalt.md")
	if !strings.Contains(content, "links:") || !strings.HasSuffix(content, "A volcanic rock.\n") {
		t.Errorf("Rocks/Basalt.md holds %q, want a links block and the body kept", content)
	}
	links := readLinks(t, v, "Rocks/Basalt.md")
	jump, ok := findLink(links, "Granite", domain.RoleJump)
	if !ok || jump.Label != "cousin" || jump.Why != "both are igneous" {
		t.Errorf("links of Rocks/Basalt.md = %+v, want a jump to Granite labelled cousin with its note", links)
	}
	if parent, ok := findLink(links, "Minerals", domain.RoleParent); !ok || parent.Type != "source" {
		t.Errorf("links of Rocks/Basalt.md = %+v, want a parent link to Minerals of type source", links)
	}
	if _, ok := findLink(readLinks(t, v, "Minerals.md"), "Granite", domain.RoleChild); !ok {
		t.Errorf("links of Minerals.md = %+v, want a child link to Granite", readLinks(t, v, "Minerals.md"))
	}
}

func TestLinkAddWithAnOldFingerprintSkipsOnlyThatNote(t *testing.T) {
	v, registry := newGeologyVault(t)
	oldFingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", "---\ntitle: Basalt\n---\nA volcanic rock, changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")
	minerals := readFingerprint(t, registry, "Minerals.md")

	arguments := fmt.Sprintf(`{"links":[
		{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Granite","role":"jump"},
		{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Minerals","role":"parent"},
		{"from":"Minerals.md","fingerprint":%q,"to":"Granite","role":"child"}]}`, oldFingerprint, oldFingerprint, minerals)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
	if len(out.Added) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(out.Added), out.Added)
	}
	if out.Added[0].Error == "" || out.Added[1].Error == "" {
		t.Errorf("results = %+v, want an error for each link into Rocks/Basalt.md", out.Added)
	}
	if out.Added[2].Error != "" {
		t.Errorf("result of Minerals.md = %+v, want it written", out.Added[2])
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
	if _, ok := findLink(readLinks(t, v, "Minerals.md"), "Granite", domain.RoleChild); !ok {
		t.Errorf("links of Minerals.md = %+v, want a child link to Granite", readLinks(t, v, "Minerals.md"))
	}
}

func TestLinkAddWithTwoFingerprintsForOneNoteRefusesBothLinks(t *testing.T) {
	for order, isOtherFirst := range map[string]bool{"current first": false, "other first": true} {
		t.Run(order, func(t *testing.T) {
			v, registry := newGeologyVault(t)
			current := readFingerprint(t, registry, "Rocks/Basalt.md")
			minerals := readFingerprint(t, registry, "Minerals.md")
			before, _ := v.Content("Rocks/Basalt.md")
			first, second := current, "1-1"
			if isOtherFirst {
				first, second = second, current
			}

			arguments := fmt.Sprintf(`{"links":[
				{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Granite","role":"jump"},
				{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Minerals","role":"parent"},
				{"from":"Minerals.md","fingerprint":%q,"to":"Granite","role":"child"}]}`, first, second, minerals)
			var out linkAddOutput
			decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
			if len(out.Added) != 3 {
				t.Fatalf("got %d results, want 3: %+v", len(out.Added), out.Added)
			}
			if out.Added[0].Error == "" || out.Added[1].Error == "" {
				t.Errorf("results = %+v, want an error for each link into Rocks/Basalt.md", out.Added)
			}
			if out.Added[2].Error != "" {
				t.Errorf("result of Minerals.md = %+v, want it written", out.Added[2])
			}
			requireContent(t, v, "Rocks/Basalt.md", before)
			if _, ok := findLink(readLinks(t, v, "Minerals.md"), "Granite", domain.RoleChild); !ok {
				t.Errorf("links of Minerals.md = %+v, want a child link to Granite", readLinks(t, v, "Minerals.md"))
			}
		})
	}
}

func TestLinkAddReportsAnUnknownRoleOrNoTargetPerLink(t *testing.T) {
	v, registry := newGeologyVault(t)
	basalt := readFingerprint(t, registry, "Rocks/Basalt.md")
	minerals := readFingerprint(t, registry, "Minerals.md")
	granite := readFingerprint(t, registry, "Rocks/Granite.md")
	mineralsBefore, _ := v.Content("Minerals.md")
	graniteBefore, _ := v.Content("Rocks/Granite.md")

	arguments := fmt.Sprintf(`{"links":[
		{"from":"Minerals.md","fingerprint":%q,"to":"Granite","role":"sideways"},
		{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Granite","role":"jump"},
		{"from":"Rocks/Granite.md","fingerprint":%q,"to":"","role":"jump"}]}`, minerals, basalt, granite)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
	if len(out.Added) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(out.Added), out.Added)
	}
	if out.Added[0].Error == "" {
		t.Errorf("result of the unknown role = %+v, want an error", out.Added[0])
	}
	if out.Added[1].Error != "" {
		t.Errorf("result of Rocks/Basalt.md = %+v, want it written", out.Added[1])
	}
	if out.Added[2].Error == "" {
		t.Errorf("result of the empty target = %+v, want an error", out.Added[2])
	}
	requireContent(t, v, "Minerals.md", mineralsBefore)
	requireContent(t, v, "Rocks/Granite.md", graniteBefore)
	if _, ok := findLink(readLinks(t, v, "Rocks/Basalt.md"), "Granite", domain.RoleJump); !ok {
		t.Errorf("links of Rocks/Basalt.md = %+v, want a jump to Granite", readLinks(t, v, "Rocks/Basalt.md"))
	}
}

func TestLinkAddWithoutAFingerprintWritesNothing(t *testing.T) {
	v, registry := newGeologyVault(t)
	before, _ := v.Content("Rocks/Basalt.md")

	arguments := `{"links":[{"from":"Rocks/Basalt.md","to":"Granite","role":"jump"}]}`
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: geologyVault, Name: "link_add", Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call link_add: %v", err)
	}
	if res.GetError() == tools.ErrorCodeUnknownTool {
		t.Fatal("call link_add: the tool is unknown")
	}
	isRefused := res.GetResult().GetIsRefusal() || res.GetError() != tools.ErrorCodeUnspecified
	if !isRefused {
		var out linkAddOutput
		decodeInto(t, res.GetResult().GetStructured(), &out)
		if len(out.Added) != 1 || out.Added[0].Error == "" {
			t.Errorf("result = %+v, want the call refused or the link reported with an error", out.Added)
		}
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkAddOfMoreThanFiftyLinksIsRefused(t *testing.T) {
	_, registry := newGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	links := make([]map[string]string, 51)
	for i := range links {
		links[i] = map[string]string{
			"from": "Rocks/Basalt.md", "fingerprint": fingerprint, "to": fmt.Sprintf("Pebble %d", i), "role": "jump",
		}
	}
	raw, err := json.Marshal(map[string]any{"links": links})
	if err != nil {
		t.Fatal(err)
	}
	requireRefusal(t, registry, geologyVault, "link_add", string(raw))

	raw, err = json.Marshal(map[string]any{"links": links[:50]})
	if err != nil {
		t.Fatal(err)
	}
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", string(raw)), &out)
	if len(out.Added) != 50 {
		t.Errorf("got %d results, want 50", len(out.Added))
	}
}

func TestLinkUpdateChangesTheLabelAndKeepsTheRoleLeftOut(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	var out linkEditOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","to":"Minerals","fingerprint":%q,"label":"made of"}`, fingerprint)), &out)
	if out.Path != "Rocks/Basalt.md" || out.Fingerprint == "" {
		t.Errorf("result = %+v, want Rocks/Basalt.md with a fingerprint", out)
	}
	if now := readFingerprint(t, registry, "Rocks/Basalt.md"); now != out.Fingerprint {
		t.Errorf("fingerprint of Rocks/Basalt.md = %q, want the one returned, %q", now, out.Fingerprint)
	}
	links := readLinks(t, v, "Rocks/Basalt.md")
	if parent, ok := findLink(links, "Minerals", domain.RoleParent); !ok || parent.Label != "made of" {
		t.Errorf("links = %+v, want the parent link to Minerals labelled made of", links)
	}
	if len(links) != 3 {
		t.Errorf("got %d links, want 3: %+v", len(links), links)
	}
}

func TestLinkUpdateKeepsTheFieldsLeftOut(t *testing.T) {
	v, registry := newGeologyVault(t)
	v.File("Rocks/Basalt.md", "---\ntitle: Basalt\nlinks:\n"+
		"  - to: Granite\n    role: jump\n    type: contrast\n    label: cousin\n    note: both are igneous\n"+
		"---\nA volcanic rock.\n")
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	callTool(t, registry, geologyVault, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","to":"Granite","fingerprint":%q,"role":"ref"}`, fingerprint))
	links := readLinks(t, v, "Rocks/Basalt.md")
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1: %+v", len(links), links)
	}
	got := links[0]
	if got.Target.Value != "Granite" || got.Role != domain.RoleRef {
		t.Errorf("link = %+v, want a ref to Granite", got)
	}
	if got.Type != "contrast" || got.Label != "cousin" || got.Why != "both are igneous" {
		t.Errorf("link = %+v, want type contrast, label cousin and note both are igneous kept", got)
	}
}

func TestLinkUpdateWithAnOldFingerprintIsRefused(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	oldFingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	content, _ := v.Content("Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", content+"Changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")

	requireRefusal(t, registry, geologyVault, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","to":"Minerals","fingerprint":%q,"label":"made of"}`, oldFingerprint))
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkUpdateWithMissingOrUnknownFieldsIsRefused(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	before, _ := v.Content("Rocks/Basalt.md")

	for _, arguments := range []string{
		`{"from":"Rocks/Basalt.md","to":"Minerals","label":"made of"}`,
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","to":"Fossils","fingerprint":%q,"label":"none"}`, fingerprint),
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","to":"Minerals","fingerprint":%q,"role":"sideways"}`, fingerprint),
	} {
		requireRefusal(t, registry, geologyVault, "link_update", arguments)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkRemoveRemovesEveryLinkToTheTarget(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	var out linkEditOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","to":"Granite","fingerprint":%q}`, fingerprint)), &out)
	if out.Path != "Rocks/Basalt.md" || out.Fingerprint == "" {
		t.Errorf("result = %+v, want Rocks/Basalt.md with a fingerprint", out)
	}
	if now := readFingerprint(t, registry, "Rocks/Basalt.md"); now != out.Fingerprint {
		t.Errorf("fingerprint of Rocks/Basalt.md = %q, want the one returned, %q", now, out.Fingerprint)
	}
	links := readLinks(t, v, "Rocks/Basalt.md")
	if len(links) != 1 || links[0].Target.Value != "Minerals" {
		t.Errorf("links = %+v, want only the parent link to Minerals", links)
	}
}

func TestLinkRemoveWithARoleRemovesOnlyTheLinkOfThatRole(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	callTool(t, registry, geologyVault, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","to":"Granite","fingerprint":%q,"role":"ref"}`, fingerprint))
	links := readLinks(t, v, "Rocks/Basalt.md")
	if _, ok := findLink(links, "Granite", domain.RoleRef); ok {
		t.Errorf("links = %+v, want the ref to Granite removed", links)
	}
	if _, ok := findLink(links, "Granite", domain.RoleJump); !ok {
		t.Errorf("links = %+v, want the jump to Granite kept", links)
	}
	if len(links) != 2 {
		t.Errorf("got %d links, want 2: %+v", len(links), links)
	}
}

func TestLinkRemoveWithAnOldFingerprintIsRefused(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	oldFingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	content, _ := v.Content("Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", content+"Changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")

	requireRefusal(t, registry, geologyVault, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","to":"Granite","fingerprint":%q}`, oldFingerprint))
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkRemoveWithoutAFingerprintOrAMatchingLinkIsRefused(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	before, _ := v.Content("Rocks/Basalt.md")

	for _, arguments := range []string{
		`{"from":"Rocks/Basalt.md","to":"Granite"}`,
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","to":"Fossils","fingerprint":%q}`, fingerprint),
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","to":"Granite","fingerprint":%q,"role":"child"}`, fingerprint),
	} {
		requireRefusal(t, registry, geologyVault, "link_remove", arguments)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}
