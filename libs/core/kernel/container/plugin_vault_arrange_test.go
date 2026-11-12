package container_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

func getRivers() map[string]string {
	return map[string]string{
		"Danube.md":    "# Danube\n\nA river of central Europe.\n",
		"Deltas.md":    "---\nlinks:\n  - to: Danube\n    role: parent\n---\n# Deltas\n",
		"Estuaries.md": "---\nlinks:\n  - to: Danube\n    role: parent\n---\n# Estuaries\n",
	}
}

func openScannedPluginVault(t *testing.T, notes map[string]string, hooks container.PluginVaultHooks) (domain.Vault, *vaultapi.Access) {
	t.Helper()
	v, current, assembly := openPluginVault(t, notes)
	shared, err := assembly.OpenPluginVault(current, hooks)
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(notes))
	for path := range notes {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	if err := shared.Writer.Scan(t.Context(), paths); err != nil {
		t.Fatal(err)
	}
	return v, shared
}

func readVaultFile(t *testing.T, v domain.Vault, path string) (string, bool) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(content), true
}

func getLinkTargets(t *testing.T, shared *vaultapi.Access, from string) []string {
	t.Helper()
	links, err := shared.Links.GetLinks(t.Context(), from)
	if err != nil {
		t.Fatalf("links of %s: %v", from, err)
	}
	targets := make([]string, 0, len(links))
	for _, link := range links {
		targets = append(targets, link.To)
	}
	return targets
}

func TestAPluginRenameMovesTheFileAndRepairsTheLinksByName(t *testing.T) {
	v, shared := openScannedPluginVault(t, getRivers(), container.PluginVaultHooks{})

	to, isFilenamed, moved, err := shared.Arranger.RenameNote(t.Context(), "Danube.md", "Donau")
	if err != nil {
		t.Fatalf("the rename failed: %v", err)
	}

	if to != "Donau.md" || !isFilenamed {
		t.Errorf("the rename returned path %q, filenamed %v; want Donau.md, true", to, isFilenamed)
	}
	if moved == nil {
		t.Fatal("the rename returned no move, and the file moved")
	}
	repaired := slices.Sorted(slices.Values(moved.Repaired))
	if moved.From != "Danube.md" || moved.To != "Donau.md" || !slices.Equal(repaired, []string{"Deltas.md", "Estuaries.md"}) {
		t.Errorf("the move was %+v, want Danube.md to Donau.md with Deltas.md and Estuaries.md repaired", *moved)
	}
	if _, found := readVaultFile(t, v, "Danube.md"); found {
		t.Error("Danube.md is still on the disk")
	}
	if _, found := readVaultFile(t, v, "Donau.md"); !found {
		t.Error("Donau.md is not on the disk")
	}
	for _, from := range []string{"Deltas.md", "Estuaries.md"} {
		if content, _ := readVaultFile(t, v, from); !strings.Contains(content, "to: Donau") {
			t.Errorf("%s does not link to Donau:\n%s", from, content)
		}
		if targets := getLinkTargets(t, shared, from); !slices.Equal(targets, []string{"Donau.md"}) {
			t.Errorf("the links of %s point to %v, want Donau.md", from, targets)
		}
	}
}

func TestPluginMovePlacesNoteInFolderAndKeepsLinksByName(t *testing.T) {
	v, shared := openScannedPluginVault(t, getRivers(), container.PluginVaultHooks{})

	moved, err := shared.Arranger.MoveNote(t.Context(), "Danube.md", "Rivers/Danube.md")
	if err != nil {
		t.Fatalf("the move failed: %v", err)
	}

	if moved.From != "Danube.md" || moved.To != "Rivers/Danube.md" || !moved.IsMoved {
		t.Errorf("the move was %+v, want Danube.md to Rivers/Danube.md", moved)
	}
	if len(moved.Dangling) != 0 {
		t.Errorf("the move left %v dangling", moved.Dangling)
	}
	if _, found := readVaultFile(t, v, "Rivers/Danube.md"); !found {
		t.Error("Rivers/Danube.md is not on the disk")
	}
	if _, found := readVaultFile(t, v, "Danube.md"); found {
		t.Error("Danube.md is still on the disk")
	}
	if targets := getLinkTargets(t, shared, "Deltas.md"); !slices.Equal(targets, []string{"Rivers/Danube.md"}) {
		t.Errorf("the links of Deltas.md point to %v, want Rivers/Danube.md", targets)
	}
}

func TestPluginRemovalMovesNoteToTrashAndReportsDangling(t *testing.T) {
	v, shared := openScannedPluginVault(t, getRivers(), container.PluginVaultHooks{})

	trashed, dangling, err := shared.Arranger.RemoveNote(t.Context(), "Danube.md")
	if err != nil {
		t.Fatalf("the removal failed: %v", err)
	}

	if !strings.HasPrefix(trashed, ".trash/") {
		t.Errorf("the note is at %q, want a path under .trash/", trashed)
	}
	if content, found := readVaultFile(t, v, trashed); !found || !strings.Contains(content, "A river of central Europe.") {
		t.Errorf("the trash contains %q (found %v), want the note's content", content, found)
	}
	if _, found := readVaultFile(t, v, "Danube.md"); found {
		t.Error("Danube.md is still at its path")
	}
	if got := slices.Sorted(slices.Values(dangling)); !slices.Equal(got, []string{"Deltas.md", "Estuaries.md"}) {
		t.Errorf("dangling is %v, want Deltas.md and Estuaries.md", got)
	}
}

func TestAPluginRemovalOfAFolderReturnsNotANote(t *testing.T) {
	v, shared := openScannedPluginVault(t, map[string]string{
		"Rivers/Danube.md": "# Danube\n",
		"Rivers/Rhine.md":  "# Rhine\n",
	}, container.PluginVaultHooks{})

	_, _, err := shared.Arranger.RemoveNote(t.Context(), "Rivers")

	if !errors.Is(err, edit.ErrNotANote) {
		t.Errorf("removing a folder returned %v, want ErrNotANote", err)
	}
	for _, path := range []string{"Rivers/Danube.md", "Rivers/Rhine.md"} {
		if _, found := readVaultFile(t, v, path); !found {
			t.Errorf("%s is not on the disk after the folder was rejected", path)
		}
	}
}

func TestAPluginRunsTheChecksOfTheVault(t *testing.T) {
	_, shared := openScannedPluginVault(t, map[string]string{
		"Rivers/Danube.md": "# Danube\n",
		"Ships/Danube.md":  "# Danube\n",
		"Deltas.md":        "---\nlinks:\n  - to: Danube\n    role: parent\n---\n# Deltas\n",
		"Lakes.md":         "---\nlinks:\n  - to: Baikal\n    role: parent\n---\n# Lakes\n",
	}, container.PluginVaultHooks{})

	ran, found, err := shared.Arranger.GetProblems(t.Context(), nil)
	if err != nil {
		t.Fatalf("the checks failed: %v", err)
	}
	if want := []domain.Check{domain.CheckParse, domain.CheckFrontmatter, domain.CheckAmbiguous}; !slices.Equal(slices.Sorted(slices.Values(ran)), slices.Sorted(slices.Values(want))) {
		t.Errorf("the checks that ran were %v, want %v", ran, want)
	}
	var ambiguous []domain.VaultProblem
	for _, problem := range found {
		switch problem.Kind {
		case domain.CheckAmbiguous:
			ambiguous = append(ambiguous, problem)
		case domain.CheckDangling:
			t.Errorf("dangling was found without being named: %+v", problem)
		}
	}
	if len(ambiguous) != 1 || ambiguous[0].Path != "Deltas.md" ||
		!slices.Equal(slices.Sorted(slices.Values(ambiguous[0].Candidates)), []string{"Rivers/Danube.md", "Ships/Danube.md"}) {
		t.Errorf("the ambiguous problems were %+v, want one against Deltas.md with both Danube notes", ambiguous)
	}

	ran, found, err = shared.Arranger.GetProblems(t.Context(), []domain.Check{domain.CheckDangling})
	if err != nil {
		t.Fatalf("the dangling check failed: %v", err)
	}
	if !slices.Equal(ran, []domain.Check{domain.CheckDangling}) {
		t.Errorf("the checks that ran were %v, want dangling", ran)
	}
	if len(found) != 1 || found[0].Path != "Lakes.md" || found[0].Kind != domain.CheckDangling || found[0].Target.Value != "Baikal" {
		t.Errorf("the dangling problems were %+v, want one against Lakes.md for Baikal", found)
	}
}

func TestPluginRenameToExistingFilenameKeepsNoteUnchanged(t *testing.T) {
	const danube = "---\ntitle: Danube\n---\n# Danube\n\nA river of central Europe.\n"
	v, shared := openScannedPluginVault(t, map[string]string{
		"Rivers/Danube.md": danube,
		"Rivers/Deltas.md": "# Deltas\n",
	}, container.PluginVaultHooks{})

	_, _, _, err := shared.Arranger.RenameNote(t.Context(), "Rivers/Danube.md", "Deltas")

	if !errors.Is(err, vault.ErrAlreadyExists) {
		t.Errorf("the rename onto Rivers/Deltas.md returned %v, want ErrAlreadyExists", err)
	}
	if content, found := readVaultFile(t, v, "Rivers/Danube.md"); !found || content != danube {
		t.Errorf("Rivers/Danube.md contains %q (found %v), want it unchanged:\n%q", content, found, danube)
	}
	if content, _ := readVaultFile(t, v, "Rivers/Deltas.md"); content != "# Deltas\n" {
		t.Errorf("Rivers/Deltas.md contains %q, want it unchanged", content)
	}
}
