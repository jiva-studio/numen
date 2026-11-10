package service_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// linkUpdateCall is one call of UpdateLinks as the host received it.
type linkUpdateCall struct {
	From     string
	To       domain.Address
	Change   domain.Link
	Baseline domain.FileStat
}

// linkRemoveCall is one call of RemoveLinks as the host received it.
type linkRemoveCall struct {
	From     string
	To       domain.Address
	Role     domain.LinkRole
	Baseline domain.FileStat
}

// linkCalls is what the host of the linked geology vault received.
type linkCalls struct {
	mu        sync.Mutex
	additions [][]host.LinkAddition
	updates   []linkUpdateCall
	removals  []linkRemoveCall
	writes    []string
}

// linkHostVault is the geology vault whose host edits the links itself, and records every call of
// its LinkEditor and every Write.
type linkHostVault struct {
	*hosttest.Vault
	calls *linkCalls
}

func (l linkHostVault) With(domain.VaultID) linkHostVault { return l }

func (l linkHostVault) AddLinks(ctx context.Context, additions []host.LinkAddition) ([]host.LinkAdded, error) {
	l.calls.mu.Lock()
	l.calls.additions = append(l.calls.additions, additions)
	l.calls.mu.Unlock()
	return l.Vault.AddLinks(ctx, additions)
}

func (l linkHostVault) UpdateLinks(
	ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat,
) (domain.FileStat, error) {
	l.calls.mu.Lock()
	l.calls.updates = append(l.calls.updates, linkUpdateCall{From: from, To: to, Change: change, Baseline: baseline})
	l.calls.mu.Unlock()
	return l.Vault.UpdateLinks(ctx, from, to, change, baseline)
}

func (l linkHostVault) RemoveLinks(
	ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat,
) (domain.FileStat, error) {
	l.calls.mu.Lock()
	l.calls.removals = append(l.calls.removals, linkRemoveCall{From: from, To: to, Role: role, Baseline: baseline})
	l.calls.mu.Unlock()
	return l.Vault.RemoveLinks(ctx, from, to, role, baseline)
}

func (l linkHostVault) Write(ctx context.Context, filePath string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	l.calls.mu.Lock()
	l.calls.writes = append(l.calls.writes, filePath)
	l.calls.mu.Unlock()
	return l.Vault.Write(ctx, filePath, content, baseline)
}

func newLinkHostVault(t *testing.T) (*hosttest.Vault, *linkCalls, *tools.Registry) {
	t.Helper()
	v, _ := newLinkedGeologyVault(t)
	calls := &linkCalls{}
	registry, err := service.New(linkHostVault{Vault: v, calls: calls}.vaults()).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return v, calls, registry
}

func statOf(t *testing.T, v *hosttest.Vault, notePath string) domain.FileStat {
	t.Helper()
	stat, err := v.StatFile(t.Context(), notePath)
	if err != nil {
		t.Fatalf("stat %s: %v", notePath, err)
	}
	return stat
}

func isSameBaseline(got, want domain.FileStat) bool {
	return got.Size == want.Size && got.ModTime.Equal(want.ModTime)
}

func getErrorText(t *testing.T, registry *tools.Registry, name, arguments string) string {
	t.Helper()
	res, err := registry.InvokeTool(t.Context(), tools.Call{VaultID: geologyVault, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if !res.GetResult().GetIsError() {
		t.Fatalf("call %s %s was not rejected: %s", name, arguments, res.GetResult().GetStructured())
	}
	return res.GetResult().GetText()
}

func TestLinkAddSendsOneHostCallPerNoteAndWritesNoFile(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	basaltStat := statOf(t, v, "Rocks/Basalt.md")
	basalt := readFingerprint(t, registry, "Rocks/Basalt.md")
	minerals := readFingerprint(t, registry, "Minerals.md")

	arguments := fmt.Sprintf(`{"links":[
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Fossils","role":"jump","type":"source","label":"older","note":"shells in the lava"},
		{"from":"Minerals.md","fingerprint":%q,"address":"Granite","role":"child"}]}`, basalt, minerals)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)

	if len(calls.writes) != 0 {
		t.Errorf("the plugin wrote %v itself, want every link written by the host", calls.writes)
	}
	if len(calls.additions) != 1 || len(calls.additions[0]) != 2 {
		t.Fatalf("the host received the additions %+v, want one call with the two links", calls.additions)
	}
	first := calls.additions[0][0]
	if first.From != "Rocks/Basalt.md" || !isSameBaseline(first.Baseline, basaltStat) {
		t.Errorf("the first addition is from %s against %+v, want Rocks/Basalt.md against %+v", first.From, first.Baseline, basaltStat)
	}
	if l := first.Link; l.Target.Value != "Fossils" || l.Role != domain.RoleJump || l.Type != "source" || l.Label != "older" || l.Why != "shells in the lava" {
		t.Errorf("the first link is %+v, want a jump to Fossils of type source labeled older with its note", l)
	}
	if second := calls.additions[0][1]; second.From != "Minerals.md" || second.Link.Target.Value != "Granite" || second.Link.Role != domain.RoleChild {
		t.Errorf("the second addition is %+v, want a child link from Minerals.md to Granite", second)
	}
	if len(out.Results) != 2 || out.Results[0].Error != "" || out.Results[1].Error != "" ||
		out.Results[0].Fingerprint == "" || out.Results[1].Fingerprint == "" {
		t.Fatalf("results = %+v, want both links written with a fingerprint", out.Results)
	}
	if now := readFingerprint(t, registry, "Rocks/Basalt.md"); now != out.Results[0].Fingerprint {
		t.Errorf("fingerprint of Rocks/Basalt.md = %q, want the one returned, %q", now, out.Results[0].Fingerprint)
	}
	if _, ok := findLink(readLinks(t, v, "Rocks/Basalt.md"), "Fossils", domain.RoleJump); !ok {
		t.Errorf("links of Rocks/Basalt.md = %+v, want a jump to Fossils", readLinks(t, v, "Rocks/Basalt.md"))
	}
}

func TestLinkAddReportsStaleNotePerLinkAndWritesTheOthers(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	oldFingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	content, _ := v.Content("Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", content+"Changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")
	minerals := readFingerprint(t, registry, "Minerals.md")

	arguments := fmt.Sprintf(`{"links":[
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Fossils","role":"jump"},
		{"from":"Minerals.md","fingerprint":%q,"address":"Granite","role":"child"}]}`, oldFingerprint, minerals)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)

	if len(calls.additions) != 1 {
		t.Fatalf("the host received %d calls of AddLinks, want 1", len(calls.additions))
	}
	if len(out.Results) != 2 || !strings.Contains(out.Results[0].Error, "Rocks/Basalt.md") || out.Results[0].Fingerprint != "" {
		t.Errorf("results = %+v, want the stale Rocks/Basalt.md rejected with an error that names it", out.Results)
	}
	if out.Results[1].Error != "" {
		t.Errorf("result of Minerals.md = %+v, want it written", out.Results[1])
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkAddRejectsAnUnknownRoleBeforeCallingTheHost(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	basalt := readFingerprint(t, registry, "Rocks/Basalt.md")
	minerals := readFingerprint(t, registry, "Minerals.md")

	arguments := fmt.Sprintf(`{"links":[
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"Fossils","role":"sideways"},
		{"from":"Minerals.md","fingerprint":%q,"address":"Granite","role":"child"}]}`, basalt, minerals)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)

	if len(out.Results) != 2 || !strings.Contains(out.Results[0].Error, "sideways") {
		t.Errorf("results = %+v, want the unknown role named in the error of the first link", out.Results)
	}
	for _, batch := range calls.additions {
		for _, addition := range batch {
			if addition.Link.Role == "sideways" {
				t.Errorf("the host received the link of the unknown role: %+v", addition)
			}
		}
	}
	if _, ok := findLink(readLinks(t, v, "Minerals.md"), "Granite", domain.RoleChild); !ok {
		t.Errorf("links of Minerals.md = %+v, want a child link to Granite", readLinks(t, v, "Minerals.md"))
	}
}

func TestLinkAddSendsAnEmptyFromOrToToTheHostAndReturnsItsErrors(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	basalt := readFingerprint(t, registry, "Rocks/Basalt.md")
	before, _ := v.Content("Rocks/Basalt.md")

	arguments := fmt.Sprintf(`{"links":[
		{"from":"","fingerprint":%q,"address":"Fossils","role":"jump"},
		{"from":"Rocks/Basalt.md","fingerprint":%q,"address":"","role":"jump"}]}`, basalt, basalt)
	var out linkAddOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_add", arguments), &out)

	if len(calls.additions) != 1 || len(calls.additions[0]) != 2 {
		t.Fatalf("the host received the additions %+v, want one call with both links", calls.additions)
	}
	if len(out.Results) != 2 {
		t.Fatalf("results = %+v, want 2", out.Results)
	}
	if want := "the path is outside the vault: it is empty"; !strings.Contains(out.Results[0].Error, want) {
		t.Errorf("result of the empty from = %+v, want the host's error %q", out.Results[0], want)
	}
	if want := "a link needs somewhere to go"; !strings.Contains(out.Results[1].Error, want) {
		t.Errorf("result of the empty to = %+v, want the host's error %q", out.Results[1], want)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkUpdateSendsTheChangeToTheHostAndWritesNoFile(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	stat := statOf(t, v, "Rocks/Basalt.md")
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	var out linkEditOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Minerals","fingerprint":%q,"role":"ref","type":"source","label":"made of","note":"its crystals"}`,
		fingerprint)), &out)

	if len(calls.writes) != 0 {
		t.Errorf("the plugin wrote %v itself, want the link updated by the host", calls.writes)
	}
	if len(calls.updates) != 1 {
		t.Fatalf("the host received %d calls of UpdateLinks, want 1", len(calls.updates))
	}
	got := calls.updates[0]
	if got.From != "Rocks/Basalt.md" || got.To.Value != "Minerals" || !isSameBaseline(got.Baseline, stat) {
		t.Errorf("UpdateLinks got from %s to %+v against %+v, want Rocks/Basalt.md to Minerals against %+v", got.From, got.To, got.Baseline, stat)
	}
	if c := got.Change; c.Role != domain.RoleRef || c.Type != "source" || c.Label != "made of" || c.Why != "its crystals" {
		t.Errorf("the change is %+v, want role ref, type source, label made of and its note", c)
	}
	if out.Path != "Rocks/Basalt.md" || out.Fingerprint == "" {
		t.Errorf("result = %+v, want Rocks/Basalt.md with a fingerprint", out)
	}
	if now := readFingerprint(t, registry, "Rocks/Basalt.md"); now != out.Fingerprint {
		t.Errorf("fingerprint of Rocks/Basalt.md = %q, want the one returned, %q", now, out.Fingerprint)
	}
	if ref, ok := findLink(readLinks(t, v, "Rocks/Basalt.md"), "Minerals", domain.RoleRef); !ok || ref.Label != "made of" {
		t.Errorf("links = %+v, want a ref to Minerals labeled made of", readLinks(t, v, "Rocks/Basalt.md"))
	}
}

func TestLinkUpdateOfAStaleNoteIsRejectedWithThePath(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	oldFingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	content, _ := v.Content("Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", content+"Changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")

	failure := getErrorText(t, registry, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Minerals","fingerprint":%q,"label":"made of"}`, oldFingerprint))
	if !strings.Contains(failure, "Rocks/Basalt.md") || !strings.Contains(failure, "link_update") {
		t.Errorf("the error is %q, want the tool and Rocks/Basalt.md named", failure)
	}
	if len(calls.updates) != 1 {
		t.Errorf("the host received %d calls of UpdateLinks, want the stale baseline checked by the host", len(calls.updates))
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkUpdateRejectsAnUnknownRoleBeforeCallingTheHost(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")
	before, _ := v.Content("Rocks/Basalt.md")

	failure := getErrorText(t, registry, "link_update", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Minerals","fingerprint":%q,"role":"sideways"}`, fingerprint))
	if !strings.Contains(failure, "sideways") {
		t.Errorf("the error is %q, want the unknown role named", failure)
	}
	if len(calls.updates) != 0 {
		t.Errorf("the host received %+v, want no call for an unknown role", calls.updates)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkRemoveSendsTargetAndRoleToTheHostAndWritesNoFile(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	stat := statOf(t, v, "Rocks/Basalt.md")
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	var out linkEditOutput
	decodeInto(t, callTool(t, registry, geologyVault, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q,"role":"ref"}`, fingerprint)), &out)

	if len(calls.writes) != 0 {
		t.Errorf("the plugin wrote %v itself, want the link removed by the host", calls.writes)
	}
	if len(calls.removals) != 1 {
		t.Fatalf("the host received %d calls of RemoveLinks, want 1", len(calls.removals))
	}
	got := calls.removals[0]
	if got.From != "Rocks/Basalt.md" || got.To.Value != "Granite" || got.Role != domain.RoleRef || !isSameBaseline(got.Baseline, stat) {
		t.Errorf("RemoveLinks got %+v, want the ref from Rocks/Basalt.md to Granite against %+v", got, stat)
	}
	if out.Path != "Rocks/Basalt.md" || out.Fingerprint == "" {
		t.Errorf("result = %+v, want Rocks/Basalt.md with a fingerprint", out)
	}
	links := readLinks(t, v, "Rocks/Basalt.md")
	if _, ok := findLink(links, "Granite", domain.RoleRef); ok {
		t.Errorf("links = %+v, want the ref to Granite removed", links)
	}
	if _, ok := findLink(links, "Granite", domain.RoleJump); !ok {
		t.Errorf("links = %+v, want the jump to Granite kept", links)
	}
}

func TestLinkRemoveWithoutARoleRequestsEveryRoleFromTheHost(t *testing.T) {
	v, calls, registry := newLinkHostVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	callTool(t, registry, geologyVault, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q}`, fingerprint))

	if len(calls.removals) != 1 || calls.removals[0].Role != "" {
		t.Fatalf("the host received %+v, want one call of RemoveLinks with no role", calls.removals)
	}
	if links := readLinks(t, v, "Rocks/Basalt.md"); len(links) != 1 || links[0].Target.Value != "Minerals" {
		t.Errorf("links = %+v, want only the parent link to Minerals", links)
	}
}

func TestLinkRemoveOfStaleNoteOrMissingLinkIsRejectedWithPath(t *testing.T) {
	v, _, registry := newLinkHostVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	failure := getErrorText(t, registry, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Fossils","fingerprint":%q}`, fingerprint))
	if !strings.Contains(failure, "no link to Fossils is written here") {
		t.Errorf("the error of a missing link is %q, want the host's text", failure)
	}

	content, _ := v.Content("Rocks/Basalt.md")
	v.File("Rocks/Basalt.md", content+"Changed by another program.\n")
	before, _ := v.Content("Rocks/Basalt.md")
	failure = getErrorText(t, registry, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q}`, fingerprint))
	if !strings.Contains(failure, "Rocks/Basalt.md") {
		t.Errorf("the error of a stale note is %q, want Rocks/Basalt.md named", failure)
	}
	requireContent(t, v, "Rocks/Basalt.md", before)
}

func TestLinkRemoveRejectsAnUnknownRoleBeforeCallingTheHost(t *testing.T) {
	_, calls, registry := newLinkHostVault(t)
	fingerprint := readFingerprint(t, registry, "Rocks/Basalt.md")

	failure := getErrorText(t, registry, "link_remove", fmt.Sprintf(
		`{"from":"Rocks/Basalt.md","address":"Granite","fingerprint":%q,"role":"sideways"}`, fingerprint))
	if !strings.Contains(failure, "sideways") {
		t.Errorf("the error is %q, want the unknown role named", failure)
	}
	if len(calls.removals) != 0 {
		t.Errorf("the host received %+v, want no call for an unknown role", calls.removals)
	}
}

func (l linkHostVault) vaults() func(domain.VaultID) service.Vault {
	return func(id domain.VaultID) service.Vault {
		on := l.With(id)
		return service.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Arranger: on, Editor: on, Selector: on}
	}
}
