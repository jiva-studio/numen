package hosttest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

const (
	basaltPath   = "Rocks/Basalt.md"
	mineralsPath = "Minerals.md"
	basaltBody   = "A volcanic rock.\n"
)

func newRockVault(basaltFrontmatter string) *hosttest.Vault {
	v := hosttest.New("Geology")
	v.File(basaltPath, "---\ntitle: Basalt\n"+basaltFrontmatter+"---\n"+basaltBody)
	v.File("Rocks/Granite.md", "---\ntitle: Granite\n---\nA plutonic rock.\n")
	v.File(mineralsPath, "---\ntitle: Minerals\n---\nQuartz and feldspar.\n")
	return v
}

func linksOf(t *testing.T, v *hosttest.Vault, notePath string) []domain.Link {
	t.Helper()
	content, ok := v.Content(notePath)
	if !ok {
		t.Fatalf("%s is not in the vault", notePath)
	}
	return markdown.Parse(domain.FileStat{Path: notePath}, []byte(content)).Links
}

func findRockLink(links []domain.Link, to string, role domain.LinkRole) (domain.Link, bool) {
	for _, l := range links {
		if l.Target.Value == to && l.Role == role {
			return l, true
		}
	}
	return domain.Link{}, false
}

func statOf(t *testing.T, v *hosttest.Vault, notePath string) domain.FileStat {
	t.Helper()
	stat, err := v.StatFile(t.Context(), notePath)
	if err != nil {
		t.Fatal(err)
	}
	return stat
}

func nameAddress(name string) domain.Address {
	return domain.Address{Scheme: domain.SchemeName, Value: name}
}

func makeNoteStale(t *testing.T, v *hosttest.Vault, notePath string) {
	t.Helper()
	content, stat, err := v.Read(t.Context(), notePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Write(t.Context(), notePath, append(content, "Seen again.\n"...), stat); err != nil {
		t.Fatal(err)
	}
}

func TestAddLinksWritesTheLinksIntoTheFrontmatterAndKeepsTheBody(t *testing.T) {
	v := newRockVault("")
	baseline := statOf(t, v, basaltPath)

	results, err := v.AddLinks(t.Context(), []host.LinkAddition{{From: basaltPath, Baseline: baseline, Link: domain.Link{
		Target: nameAddress("Granite"), Role: domain.RoleJump, Type: "relative", Label: "cousin", Why: "both are igneous",
	}}})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("the results are %+v, want one written link", results)
	}
	if now := statOf(t, v, basaltPath); !results[0].Stat.ModTime.Equal(now.ModTime) || results[0].Stat.Size != now.Size {
		t.Errorf("the result has the stat %+v, want the stat of the note now, %+v", results[0].Stat, now)
	}
	link, ok := findRockLink(linksOf(t, v, basaltPath), "Granite", domain.RoleJump)
	if !ok || link.Type != "relative" || link.Label != "cousin" || link.Why != "both are igneous" {
		t.Errorf("the links of %s are %+v, want a jump to Granite of type relative labeled cousin with its note", basaltPath, linksOf(t, v, basaltPath))
	}
	if content, _ := v.Content(basaltPath); !strings.HasSuffix(content, basaltBody) || !strings.Contains(content, "title: Basalt") {
		t.Errorf("%s contains %q, want the title and the body kept", basaltPath, content)
	}
}

func TestAddLinksWritesTheLinksOfOneNoteAgainstOneBaseline(t *testing.T) {
	v := newRockVault("")
	basalt := statOf(t, v, basaltPath)
	minerals := statOf(t, v, mineralsPath)

	results, err := v.AddLinks(t.Context(), []host.LinkAddition{
		{From: basaltPath, Baseline: basalt, Link: domain.Link{Target: nameAddress("Granite"), Role: domain.RoleJump}},
		{From: mineralsPath, Baseline: minerals, Link: domain.Link{Target: nameAddress("Basalt"), Role: domain.RoleChild}},
		{From: basaltPath, Baseline: basalt, Link: domain.Link{Target: nameAddress("Minerals"), Role: domain.RoleParent}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 3 {
		t.Fatalf("the results are %+v, want 3", results)
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("result %d is %+v, want the link written", i, r)
		}
	}
	if !results[0].Stat.ModTime.Equal(results[2].Stat.ModTime) || results[0].Stat.Size != results[2].Stat.Size {
		t.Errorf("the links of %s have the stats %+v and %+v, want one write", basaltPath, results[0].Stat, results[2].Stat)
	}
	links := linksOf(t, v, basaltPath)
	if _, ok := findRockLink(links, "Granite", domain.RoleJump); !ok {
		t.Errorf("the links of %s are %+v, want a jump to Granite", basaltPath, links)
	}
	if _, ok := findRockLink(links, "Minerals", domain.RoleParent); !ok {
		t.Errorf("the links of %s are %+v, want a parent link to Minerals", basaltPath, links)
	}
	if _, ok := findRockLink(linksOf(t, v, mineralsPath), "Basalt", domain.RoleChild); !ok {
		t.Errorf("the links of %s are %+v, want a child link to Basalt", mineralsPath, linksOf(t, v, mineralsPath))
	}
}

func TestAddLinksRejectsStaleBaselineNoteAndWritesTheOthers(t *testing.T) {
	v := newRockVault("")
	basalt := statOf(t, v, basaltPath)
	minerals := statOf(t, v, mineralsPath)
	makeNoteStale(t, v, basaltPath)
	before, _ := v.Content(basaltPath)

	results, err := v.AddLinks(t.Context(), []host.LinkAddition{
		{From: basaltPath, Baseline: basalt, Link: domain.Link{Target: nameAddress("Granite"), Role: domain.RoleJump}},
		{From: mineralsPath, Baseline: minerals, Link: domain.Link{Target: nameAddress("Basalt"), Role: domain.RoleChild}},
		{From: basaltPath, Baseline: basalt, Link: domain.Link{Target: nameAddress("Minerals"), Role: domain.RoleParent}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 3 {
		t.Fatalf("the results are %+v, want 3", results)
	}
	for _, i := range []int{0, 2} {
		if !errors.Is(results[i].Err, host.ErrStale) {
			t.Errorf("result %d is %+v, want ErrStale", i, results[i])
		}
	}
	if results[1].Err != nil {
		t.Errorf("the link into %s is %+v, want it written", mineralsPath, results[1])
	}
	if after, _ := v.Content(basaltPath); after != before {
		t.Errorf("%s contains %q after a stale write, want it unchanged: %q", basaltPath, after, before)
	}
}

const granitePair = "links:\n" +
	"  - to: Granite\n    role: jump\n    type: relative\n    label: cousin\n    note: both are igneous\n" +
	"  - to: Minerals\n    role: parent\n"

func TestUpdateLinksChangesTheFieldsGivenAndKeepsTheOthers(t *testing.T) {
	v := newRockVault(granitePair)

	stat, err := v.UpdateLinks(t.Context(), basaltPath, nameAddress("Granite"), domain.Link{Type: "sibling"}, statOf(t, v, basaltPath))
	if err != nil {
		t.Fatal(err)
	}

	link, ok := findRockLink(linksOf(t, v, basaltPath), "Granite", domain.RoleJump)
	if !ok || link.Type != "sibling" || link.Label != "cousin" || link.Why != "both are igneous" {
		t.Errorf("the links of %s are %+v, want the jump to Granite of type sibling with its label and note kept", basaltPath, linksOf(t, v, basaltPath))
	}
	if now := statOf(t, v, basaltPath); !stat.ModTime.Equal(now.ModTime) || stat.Size != now.Size {
		t.Errorf("the update returned the stat %+v, want the stat of the note now, %+v", stat, now)
	}
}

func TestUpdateLinksChangesTheRoleLabelAndNoteKeepingTheTarget(t *testing.T) {
	v := newRockVault(granitePair)

	_, err := v.UpdateLinks(t.Context(), basaltPath, nameAddress("Granite"),
		domain.Link{Role: domain.RoleRef, Label: "kin", Why: "cooled from magma"}, statOf(t, v, basaltPath))
	if err != nil {
		t.Fatal(err)
	}

	links := linksOf(t, v, basaltPath)
	link, ok := findRockLink(links, "Granite", domain.RoleRef)
	if !ok || link.Label != "kin" || link.Why != "cooled from magma" || link.Type != "relative" {
		t.Errorf("the links of %s are %+v, want a ref to Granite labeled kin with the new note and the type kept", basaltPath, links)
	}
	if _, ok := findRockLink(links, "Granite", domain.RoleJump); ok {
		t.Errorf("the links of %s are %+v, want no jump to Granite left", basaltPath, links)
	}
	if _, ok := findRockLink(links, "Minerals", domain.RoleParent); !ok {
		t.Errorf("the links of %s are %+v, want the parent link to Minerals kept", basaltPath, links)
	}
}

const graniteTwice = "links:\n" +
	"  - to: Granite\n    role: jump\n" +
	"  - to: Granite\n    role: ref\n" +
	"  - to: Minerals\n    role: parent\n"

func TestRemoveLinksRemovesEveryLinkToTheTarget(t *testing.T) {
	v := newRockVault(graniteTwice)

	if _, err := v.RemoveLinks(t.Context(), basaltPath, nameAddress("Granite"), "", statOf(t, v, basaltPath)); err != nil {
		t.Fatal(err)
	}

	links := linksOf(t, v, basaltPath)
	if len(links) != 1 || links[0].Target.Value != "Minerals" {
		t.Errorf("the links of %s are %+v, want only the parent link to Minerals", basaltPath, links)
	}
	if content, _ := v.Content(basaltPath); !strings.HasSuffix(content, basaltBody) {
		t.Errorf("%s contains %q, want the body kept", basaltPath, content)
	}
}

func TestRemoveLinksOfARoleRemovesOnlyTheLinksOfThatRole(t *testing.T) {
	v := newRockVault(graniteTwice)

	if _, err := v.RemoveLinks(t.Context(), basaltPath, nameAddress("Granite"), domain.RoleRef, statOf(t, v, basaltPath)); err != nil {
		t.Fatal(err)
	}

	links := linksOf(t, v, basaltPath)
	if _, ok := findRockLink(links, "Granite", domain.RoleRef); ok {
		t.Errorf("the links of %s are %+v, want the ref to Granite removed", basaltPath, links)
	}
	if _, ok := findRockLink(links, "Granite", domain.RoleJump); !ok || len(links) != 2 {
		t.Errorf("the links of %s are %+v, want the jump to Granite and the parent link to Minerals kept", basaltPath, links)
	}
}

func TestRemoveLinksToATargetTheNoteDoesNotLinkIsAnError(t *testing.T) {
	v := newRockVault(graniteTwice)
	before, _ := v.Content(basaltPath)

	_, err := v.RemoveLinks(t.Context(), basaltPath, nameAddress("Obsidian"), "", statOf(t, v, basaltPath))

	const want = "no link to Obsidian is written here"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("the removal returned %v, want an error containing %q", err, want)
	}
	if after, _ := v.Content(basaltPath); after != before {
		t.Errorf("%s contains %q, want it unchanged", basaltPath, after)
	}
}

func TestRemoveLinksOfARoleTheNoteDoesNotLinkWithIsAnError(t *testing.T) {
	v := newRockVault(graniteTwice)

	before, _ := v.Content(basaltPath)

	_, err := v.RemoveLinks(t.Context(), basaltPath, nameAddress("Granite"), domain.RoleChild, statOf(t, v, basaltPath))

	if err == nil || !strings.Contains(err.Error(), "no link to Granite is written here") {
		t.Errorf("the removal of a child link to Granite returned %v, want an error containing %q", err, "no link to Granite is written here")
	}
	if after, _ := v.Content(basaltPath); after != before {
		t.Errorf("%s contains %q after the rejected removal, want it unchanged", basaltPath, after)
	}
}

func TestUpdateLinksAndRemoveLinksRejectAStaleBaseline(t *testing.T) {
	v := newRockVault(granitePair)
	outdated := statOf(t, v, basaltPath)
	makeNoteStale(t, v, basaltPath)
	before, _ := v.Content(basaltPath)

	_, updateErr := v.UpdateLinks(t.Context(), basaltPath, nameAddress("Granite"), domain.Link{Label: "kin"}, outdated)
	_, removeErr := v.RemoveLinks(t.Context(), basaltPath, nameAddress("Granite"), "", outdated)

	for call, err := range map[string]error{"UpdateLinks": updateErr, "RemoveLinks": removeErr} {
		if !errors.Is(err, host.ErrStale) {
			t.Errorf("%s returned %v, want ErrStale", call, err)
		}
	}
	if after, _ := v.Content(basaltPath); after != before {
		t.Errorf("%s contains %q after stale edits, want it unchanged", basaltPath, after)
	}
}
