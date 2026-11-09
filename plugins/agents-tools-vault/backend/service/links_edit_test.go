package service_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

type linkAddItem struct {
	From        string `json:"from"`
	Address     string `json:"address"`
	Fingerprint string `json:"fingerprint"`
	Error       string `json:"error"`
}

type linkAddOutput struct {
	Results []linkAddItem `json:"results"`
}

type linkEditOutput struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

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

func readLinks(t *testing.T, v *hosttest.Vault, notePath string) []domain.Link {
	t.Helper()
	content, ok := v.Content(notePath)
	if !ok {
		t.Fatalf("%s is not in the vault", notePath)
	}
	return markdown.Parse(domain.FileStat{Path: notePath}, []byte(content)).Links
}

func findLink(links []domain.Link, to string, role domain.LinkRole) (domain.Link, bool) {
	for _, l := range links {
		if l.Target.Value == to && l.Role == role {
			return l, true
		}
	}
	return domain.Link{}, false
}

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
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Granite","role":"jump","label":"cousin","note":"both are igneous"},
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Minerals","role":"parent","type":"source"},
		{"from":"Minerals.md","fingerprint":%q,"address":"Granite","role":"child"}]}`, basalt, basalt, minerals)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
	if len(out.Results) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(out.Results), out.Results)
	}
	for i, want := range []struct{ from, to string }{
		{"Rocks/Basalt.md", "Granite"}, {"Rocks/Basalt.md", "Minerals"}, {"Minerals.md", "Granite"},
	} {
		if got := out.Results[i]; got.From != want.from || got.Address != want.to || got.Error != "" || got.Fingerprint == "" {
			t.Errorf("result %d = %+v, want %s to %s with a fingerprint and no error", i, got, want.from, want.to)
		}
	}
	if out.Results[0].Fingerprint != out.Results[1].Fingerprint {
		t.Errorf("the links into Rocks/Basalt.md have fingerprints %q and %q, want one write",
			out.Results[0].Fingerprint, out.Results[1].Fingerprint)
	}
	if now := readFingerprint(t, registry, "Rocks/Basalt.md"); now != out.Results[0].Fingerprint {
		t.Errorf("fingerprint of Rocks/Basalt.md = %q, want the one returned, %q", now, out.Results[0].Fingerprint)
	}

	content, _ := v.Content("Rocks/Basalt.md")
	if !strings.Contains(content, "links:") || !strings.HasSuffix(content, "A volcanic rock.\n") {
		t.Errorf("Rocks/Basalt.md contains %q, want a links block and the body kept", content)
	}
	links := readLinks(t, v, "Rocks/Basalt.md")
	jump, ok := findLink(links, "Granite", domain.RoleJump)
	if !ok || jump.Label != "cousin" || jump.Why != "both are igneous" {
		t.Errorf("links of Rocks/Basalt.md = %+v, want a jump to Granite labeled cousin with its note", links)
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
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Granite","role":"jump"},
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Minerals","role":"parent"},
		{"from":"Minerals.md","fingerprint":%q,"address":"Granite","role":"child"}]}`, oldFingerprint, oldFingerprint, minerals)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
	if len(out.Results) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(out.Results), out.Results)
	}
	if out.Results[0].Error == "" || out.Results[1].Error == "" {
		t.Errorf("results = %+v, want an error for each link into Rocks/Basalt.md", out.Results)
	}
	if out.Results[2].Error != "" {
		t.Errorf("result of Minerals.md = %+v, want it written", out.Results[2])
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
	if _, ok := findLink(readLinks(t, v, "Minerals.md"), "Granite", domain.RoleChild); !ok {
		t.Errorf("links of Minerals.md = %+v, want a child link to Granite", readLinks(t, v, "Minerals.md"))
	}
}

func TestLinkAddSendsTwoFingerprintsOfOneNoteToTheHost(t *testing.T) {
	for order, isOtherFirst := range map[string]bool{"current first": false, "other first": true} {
		t.Run(order, func(t *testing.T) {
			v, calls, registry := newLinkHostVault(t)
			current := readFingerprint(t, registry, "Rocks/Basalt.md")
			minerals := readFingerprint(t, registry, "Minerals.md")
			first, second := current, "1-1"
			if isOtherFirst {
				first, second = second, current
			}

			arguments := fmt.Sprintf(`{"links":[
				{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Granite","role":"jump"},
				{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Minerals","role":"child"},
				{"from":"Minerals.md","fingerprint":%q,"address":"Granite","role":"child"}]}`, first, second, minerals)
			var out linkAddOutput
			decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
			if len(calls.additions) != 1 || len(calls.additions[0]) != 3 {
				t.Fatalf("the host received the additions %+v, want one call of the three links", calls.additions)
			}
			if len(out.Results) != 3 {
				t.Fatalf("got %d results, want 3: %+v", len(out.Results), out.Results)
			}
			if out.Results[1].Error == "" {
				t.Errorf("result of the second link into Rocks/Basalt.md = %+v, want the error of the host", out.Results[1])
			}
			if out.Results[2].Error != "" {
				t.Errorf("result of Minerals.md = %+v, want it written", out.Results[2])
			}
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
		{"from":"Minerals.md","fingerprint":%q,"address":"Granite","role":"sideways"},
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Granite","role":"jump"},
		{"from":"Rocks/Granite.md","fingerprint":%q,"address":"","role":"jump"}]}`, minerals, basalt, granite)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)
	if len(out.Results) != 3 {
		t.Fatalf("got %d results, want 3: %+v", len(out.Results), out.Results)
	}
	if out.Results[0].Error == "" {
		t.Errorf("result of the unknown role = %+v, want an error", out.Results[0])
	}
	if out.Results[1].Error != "" {
		t.Errorf("result of Rocks/Basalt.md = %+v, want it written", out.Results[1])
	}
	if want := "a link needs somewhere to go"; !strings.Contains(out.Results[2].Error, want) {
		t.Errorf("result of the empty target = %+v, want the host's error %q", out.Results[2], want)
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

	arguments := `{"links":[{"from":"Rocks/Basalt.md","address":"Granite","role":"jump"}]}`
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: geologyVault, Name: "link_add", Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call link_add: %v", err)
	}
	if res.GetError() == tools.ErrorCodeUnknownTool {
		t.Fatal("call link_add: the tool is unknown")
	}
	isRefused := res.GetResult().GetIsError() || res.GetError() != tools.ErrorCodeUnspecified
	if !isRefused {
		var out linkAddOutput
		decodeInto(t, res.GetResult().GetStructured(), &out)
		if len(out.Results) != 1 || out.Results[0].Error == "" {
			t.Errorf("result = %+v, want the call rejected or the link reported with an error", out.Results)
		}
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkAddOfFiftyOneLinksSendsThemToTheHost(t *testing.T) {
	_, calls, registry := newLinkHostVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	links := make([]map[string]string, 51)
	for i := range links {
		links[i] = map[string]string{
			"from": "Rocks/Basalt.md", "fingerprint": fingerprint, "address": fmt.Sprintf("Pebble %d", i), "role": "jump",
		}
	}
	raw, err := json.Marshal(map[string]any{"links": links})
	if err != nil {
		t.Fatal(err)
	}

	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", string(raw)), &out)
	if len(calls.additions) != 1 || len(calls.additions[0]) != 51 {
		t.Errorf("the host received the additions of %d calls, want one call of 51 links", len(calls.additions))
	}
	if len(out.Results) != 51 {
		t.Errorf("got %d results, want 51", len(out.Results))
	}
}

func TestLinkToolsNameTheTargetAddress(t *testing.T) {
	_, registry := newLinkedGeologyVault(t)
	basalt := readFingerprint(t, registry, "Rocks/Basalt.md")

	answer := decodeAnswer(t, callTool(t, registry, geologyVault, "link_add", fmt.Sprintf(
		`{"links":[{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Fossils","role":"jump"}]}`, basalt)))
	results := requireList(t, "answer[results]", answer["results"])
	if len(results) != 1 {
		t.Fatalf("results = %v, want one", results)
	}
	result, isObject := results[0].(map[string]any)
	if !isObject {
		t.Fatalf("the result is %v, not an object", results[0])
	}
	requireKeys(t, "a result of link_add", result, []string{"from", "address", "fingerprint"}, []string{"to"})
	if result["address"] != "Fossils" {
		t.Errorf("the result of link_add has the address %v, want Fossils", result["address"])
	}

	requireKeys(t, "the input schema of a link of link_add", getSchemaProperties(t, getToolSchema(t, registry, "link_add", false), "links"),
		[]string{"address"}, []string{"to"})
	requireKeys(t, "the output schema of a result of link_add", getSchemaProperties(t, getToolSchema(t, registry, "link_add", true), "results"),
		[]string{"address"}, []string{"to"})
	for _, name := range []string{"link_update", "link_remove"} {
		requireKeys(t, "the input schema of "+name, getSchemaProperties(t, getToolSchema(t, registry, name, false)),
			[]string{"address"}, []string{"to"})
	}
}

func TestLinkToolsRejectTheToArgument(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	before, _ := v.Content("Rocks/Basalt.md")

	for name, arguments := range map[string]string{
		"link_add":    fmt.Sprintf(`{"links":[{"from":"Rocks/Basalt.md","fingerprint":%q,"to":"Fossils","role":"jump"}]}`, fingerprint),
		"link_update": fmt.Sprintf(`{"from":"Rocks/Basalt.md","to":"Minerals","fingerprint":%q,"label":"made of"}`, fingerprint),
		"link_remove": fmt.Sprintf(`{"from":"Rocks/Basalt.md","to":"Granite","fingerprint":%q}`, fingerprint),
	} {
		text := getRejectionText(t, registry, geologyVault, name, arguments)
		if !strings.Contains(text, "to is not known") && !strings.Contains(text, "address is missing") {
			t.Errorf("%s with the argument to gave the error %q, want to rejected as not known or address as missing", name, text)
		}
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkEditWithAnEmptyAddressIsRejected(t *testing.T) {
	_, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	for name, arguments := range map[string]string{
		"link_update": fmt.Sprintf(`{"from":"Rocks/Basalt.md","address":"","fingerprint":%q,"label":"made of"}`, fingerprint),
		"link_remove": fmt.Sprintf(`{"from":"Rocks/Basalt.md","address":"","fingerprint":%q}`, fingerprint),
	} {
		text := getRejectionText(t, registry, geologyVault, name, arguments)
		if want := name + ": address is required"; text != want {
			t.Errorf("%s with an empty address gave the error %q, want %q", name, text, want)
		}
	}
}

func TestLinkUpdateChangesTheLabelAndKeepsTheRoleLeftOut(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	var out linkEditOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Minerals","fingerprint":%q,"label":"made of"}`, fingerprint)), &out)
	if out.Path != "Rocks/Basalt.md" || out.Fingerprint == "" {
		t.Errorf("result = %+v, want Rocks/Basalt.md with a fingerprint", out)
	}
	if now := readFingerprint(t, registry, "Rocks/Basalt.md"); now != out.Fingerprint {
		t.Errorf("fingerprint of Rocks/Basalt.md = %q, want the one returned, %q", now, out.Fingerprint)
	}
	links := readLinks(t, v, "Rocks/Basalt.md")
	if parent, ok := findLink(links, "Minerals", domain.RoleParent); !ok || parent.Label != "made of" {
		t.Errorf("links = %+v, want the parent link to Minerals labeled made of", links)
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
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q,"role":"ref"}`, fingerprint))
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

func TestLinkUpdateWithAnOldFingerprintIsRejected(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	oldFingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	content, _ := v.Content("Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", content+"Changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")

	requireRejection(t, registry, geologyVault, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Minerals","fingerprint":%q,"label":"made of"}`, oldFingerprint))
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkUpdateWithMissingOrUnknownFieldsIsRejected(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	before, _ := v.Content("Rocks/Basalt.md")

	for _, arguments := range []string{
		`{"from":"Rocks/Basalt.md","address":"Minerals","label":"made of"}`,
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","address":"Fossils","fingerprint":%q,"label":"none"}`, fingerprint),
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","address":"Minerals","fingerprint":%q,"role":"sideways"}`, fingerprint),
	} {
		requireRejection(t, registry, geologyVault, "link_update", arguments)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkRemoveRemovesEveryLinkToTheTarget(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	var out linkEditOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q}`, fingerprint)), &out)
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
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q,"role":"ref"}`, fingerprint))
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

func TestLinkRemoveWithAnOldFingerprintIsRejected(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	oldFingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	content, _ := v.Content("Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", content+"Changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")

	requireRejection(t, registry, geologyVault, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q}`, oldFingerprint))
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkRemoveWithoutAFingerprintOrAMatchingLinkIsRejected(t *testing.T) {
	v, registry := newLinkedGeologyVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	before, _ := v.Content("Rocks/Basalt.md")

	for _, arguments := range []string{
		`{"from":"Rocks/Basalt.md","address":"Granite"}`,
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","address":"Fossils","fingerprint":%q}`, fingerprint),
		fmt.Sprintf(`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q,"role":"child"}`, fingerprint),
	} {
		requireRejection(t, registry, geologyVault, "link_remove", arguments)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}
