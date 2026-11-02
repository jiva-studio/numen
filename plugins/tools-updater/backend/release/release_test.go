package release_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-updater/backend/release"
)

func TestParseBuildReadsPlainNumbersAndStamps(t *testing.T) {
	cases := map[string]release.Build{
		"920":                       920,
		"2026.10.1-beta.920":        920,
		"0.4.0-alpha.1-364-283d1d5": 364,
		"2026.10.2-144-ab260c4":     144,
		"2026.10.2":                 0,
		"0.0.0-dev":                 0,
		"0":                         0,
		"invalid":                   0,
		"":                          0,
	}
	for text, want := range cases {
		if got := release.ParseBuild(text); got != want {
			t.Errorf("ParseBuild(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestParseBuildIgnoresSurroundingSpace(t *testing.T) {
	if got := release.ParseBuild("  45 "); got != 45 {
		t.Errorf("ParseBuild = %d, want 45", got)
	}
}

func TestReadAcceptsTheBuildAsNumberOrText(t *testing.T) {
	cases := map[string]release.Build{
		`{"version":"2026.10.1","build":950}`:                       950,
		`{"version":"2026.10.1","build":"950"}`:                     950,
		`{"version":"2026.10.1-beta.7","build":"2026.10.1-beta.7"}`: 7,
		`{"version":"2026.10.1","build":null}`:                      0,
		`{"version":"2026.10.1","build":true}`:                      0,
		`{"version":"2026.10.1","build":9.5}`:                       0,
		`{"version":"2026.10.1"}`:                                   0,
	}
	for body, want := range cases {
		manifest, err := release.Read([]byte(body))
		if err != nil {
			t.Fatalf("Read(%s): %v", body, err)
		}
		if manifest.Build != want {
			t.Errorf("Read(%s) build = %d, want %d", body, manifest.Build, want)
		}
	}
}

func TestReadFailsOnTextThatIsNoJSON(t *testing.T) {
	if _, err := release.Read([]byte("<html>")); err == nil {
		t.Error("Read of a page that is no JSON succeeded")
	}
}

func TestIsNewerComparesBuildsWhereTheRunningBuildHasANumber(t *testing.T) {
	cases := []struct {
		name    string
		current release.Current
		latest  release.Manifest
		want    bool
	}{
		{"a greater build", release.Current{Version: "a", Build: 100}, release.Manifest{Version: "b", Build: 105}, true},
		{"the same build", release.Current{Version: "a", Build: 100}, release.Manifest{Version: "b", Build: 100}, false},
		{"a lesser build", release.Current{Version: "a", Build: 105}, release.Manifest{Version: "b", Build: 100}, false},
		{"the same version", release.Current{Version: "2026.10.1", Build: 1}, release.Manifest{Version: "2026.10.1", Build: 500}, false},
		{"a latest with no build", release.Current{Version: "a", Build: 100}, release.Manifest{Version: "b", Build: 0}, false},
	}
	for _, c := range cases {
		if got := release.IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("%s: IsNewer = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestIsNewerComparesVersionsWhereTheRunningBuildHasNoNumber(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"a development build", "0.0.0-dev", "2026.10.1", true},
		{"an older release", "2026.9.0", "2026.10.1", true},
		{"a newer release", "2026.11.0", "2026.10.1", false},
		{"the beta of the release", "2026.10.1-beta.884", "2026.10.1", true},
		{"the release of the beta", "2026.10.1", "2026.10.1-beta.884", false},
	}
	for _, c := range cases {
		got := release.IsNewer(release.Current{Version: c.current}, release.Manifest{Version: c.latest, Build: 900})
		if got != c.want {
			t.Errorf("%s: IsNewer = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCompareVersionsOrdersReleasesAndPrereleases(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2026.10.1", "2026.10.1", 0},
		{"2026.10.2", "2026.10.1", 1},
		{"2026.9.9", "2026.10.0", -1},
		{"v1.2", "1.2.0", 0},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"1.0.0-beta.2", "1.0.0-alpha.9", 1},
		{"1.0.0-1", "1.0.0-alpha", -1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"x.y", "0.0", 0},
	}
	for _, c := range cases {
		if got := release.CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func files() []release.File {
	return []release.File{
		{Name: "numen-macos.dmg", Platform: "macos", Arch: "universal", Kind: "dmg", URL: "https://dl.numen.md/numen-macos.dmg", Size: 100, SHA256: "hash1"},
		{Name: "numen-windows-setup.exe", Platform: "windows", Arch: "amd64", Kind: "exe", URL: "https://dl.numen.md/numen-windows-setup.exe", Size: 100, SHA256: "hash2"},
		{Name: "numen-linux-amd64.deb", Platform: "linux", Arch: "amd64", Kind: "deb", URL: "https://dl.numen.md/numen-linux-amd64.deb", Size: 100, SHA256: "hash3"},
	}
}

func TestFindPlatformFilePicksTheMainFileOfThePlatform(t *testing.T) {
	cases := map[string]string{
		release.PlatformDarwin:  "numen-macos.dmg",
		release.PlatformWindows: "numen-windows-setup.exe",
		release.PlatformLinux:   "numen-linux-amd64.deb",
	}
	for platform, want := range cases {
		got := release.FindPlatformFile(files(), platform)
		if got == nil || got.Name != want {
			t.Errorf("FindPlatformFile(%s) = %v, want %s", platform, got, want)
		}
	}
}

func TestFindPlatformFileFallsBackToAnyFileOfThePlatform(t *testing.T) {
	list := []release.File{
		{Name: "numen.zip", Platform: "windows", Arch: "", Kind: "zip", URL: "u", Size: 1, SHA256: "h"},
		{Name: "numen.tar.gz", Platform: "linux", Arch: "", Kind: "tar.gz", URL: "u", Size: 1, SHA256: "h"},
		{Name: "numen.AppImage", Platform: "linux", Arch: "", Kind: "appimage", URL: "u", Size: 1, SHA256: "h"},
	}
	if got := release.FindPlatformFile(list, release.PlatformWindows); got == nil || got.Name != "numen.zip" {
		t.Errorf("windows file = %v", got)
	}
	if got := release.FindPlatformFile(list, release.PlatformLinux); got == nil || got.Name != "numen.tar.gz" {
		t.Errorf("linux file = %v", got)
	}
}

func TestFindPlatformFileFindsNothingInNoFiles(t *testing.T) {
	if got := release.FindPlatformFile(nil, release.PlatformWindows); got != nil {
		t.Errorf("file = %v, want none", got)
	}
}

func TestEvaluateOffersTheFileOfThePlatformAndTheReleasePage(t *testing.T) {
	latest := release.Manifest{
		Version: "2026.10.1", Build: 950, Commit: "abc1234", Stamp: "", Channel: "stable",
		Notes: "", Released: "2026-10-15T12:00:00Z", Files: files(),
	}

	update, ok := release.Evaluate(latest, release.Current{Version: "2026.9.0", Build: 900, Platform: release.PlatformWindows})

	if !ok {
		t.Fatal("no update offered")
	}
	if update.Version != "2026.10.1" || update.Build != 950 || update.Channel != "stable" || update.ReleasedAt != "2026-10-15T12:00:00Z" {
		t.Errorf("update = %+v", update)
	}
	if update.DownloadURL != "https://dl.numen.md/numen-windows-setup.exe" {
		t.Errorf("download = %s", update.DownloadURL)
	}
	if update.NotesURL != "https://github.com/jiva-studio/numen/releases/tag/v2026.10.1" {
		t.Errorf("notes = %s", update.NotesURL)
	}
	if update.File == nil || update.File.SHA256 != "hash2" {
		t.Errorf("file = %v", update.File)
	}
}

func TestEvaluateSendsLinuxToTheDownloadPage(t *testing.T) {
	latest := release.Manifest{Version: "2026.10.1", Build: 950, Files: files()}

	update, ok := release.Evaluate(latest, release.Current{Version: "2026.9.0", Build: 900, Platform: release.PlatformLinux})

	if !ok || update.DownloadURL != release.DownloadPage {
		t.Errorf("update = %+v, %v", update, ok)
	}
}

func TestEvaluateOffersNothingToACurrentBuild(t *testing.T) {
	latest := release.Manifest{Version: "2026.10.1", Build: 950, Files: files()}

	if _, ok := release.Evaluate(latest, release.Current{Version: "2026.10.1", Build: 950, Platform: release.PlatformLinux}); ok {
		t.Error("an update was offered to the current build")
	}
}
