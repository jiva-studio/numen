package editor

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func openAssetWindow(t *testing.T, vault domain.Vault) http.Handler {
	t.Helper()
	api := newBareServices()
	api.readFiles(vaultreader.Readers{}, nil, file.NewGetMediaType(contenttype.Open()))
	api.state.Runtime.SetVault(vault)
	return api.NewHandler(http.NotFoundHandler())
}

func askAsset(t *testing.T, handler http.Handler, name string) *httptest.ResponseRecorder {
	t.Helper()
	return ask(t, handler, "/assets/"+name)
}

func TestTheWindowServesAFileOfTheVaultAsAnAsset(t *testing.T) {
	vault := domain.Vault{ID: "v", Name: "v", Path: filepath.Join(t.TempDir(), "vault")}
	writeAssetFile(t, filepath.Join(vault.Path, "pictures", "kiln.png"), "glazed")

	got := askAsset(t, openAssetWindow(t, vault), "pictures/kiln.png")

	if got.Code != http.StatusOK || got.Body.String() != "glazed" || got.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("answered %d %q as %q", got.Code, got.Body.String(), got.Header().Get("Content-Type"))
	}
}

func TestTheWindowServesNoAssetFromOutsideTheVaultOrAFolder(t *testing.T) {
	dir := t.TempDir()
	vault := domain.Vault{ID: "v", Name: "v", Path: filepath.Join(dir, "vault")}
	writeAssetFile(t, filepath.Join(vault.Path, "notes", "a.md"), "inside")
	writeAssetFile(t, filepath.Join(dir, "vault-other", "secret.txt"), "outside")
	if err := os.Symlink(filepath.Join(dir, "vault-other"), filepath.Join(vault.Path, "door")); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}
	handler := openAssetWindow(t, vault)

	for _, name := range []string{"..%2Fvault-other%2Fsecret.txt", "door/secret.txt", "%2Fetc%2Fpasswd", "notes", "notes/missing.md"} {
		if got := askAsset(t, handler, name); got.Code != http.StatusNotFound {
			t.Errorf("%s answered %d %q, want not found", name, got.Code, got.Body.String())
		}
	}
}

func writeAssetFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAssetHeadersFollowTheExtension(t *testing.T) {
	cases := []struct {
		name, wantType string
		wantDownload   bool
	}{
		{"kiln.png", "image/png", false},
		{"KILN.PNG", "image/png", false},
		{"talk.mp3", "audio/mpeg", false},
		{"page.html", "application/octet-stream", true},
		{"script.js", "application/octet-stream", true},
		{"notes.md", "application/octet-stream", true},
		{"no-extension", "application/octet-stream", true},
	}
	types := file.NewGetMediaType(contenttype.Open())
	for _, c := range cases {
		head := http.Header{}
		setAssetHeaders(head, types.Execute(c.name))
		if got := head.Get("Content-Type"); got != c.wantType {
			t.Errorf("%s: type %q, want %q", c.name, got, c.wantType)
		}
		if got := head.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options %q, want nosniff", c.name, got)
		}
		if got := head.Get("Content-Disposition") == "attachment"; got != c.wantDownload {
			t.Errorf("%s: attachment %v, want %v", c.name, got, c.wantDownload)
		}
	}
}

func TestAnSVGIsSandboxedWhenOpenedAsAPage(t *testing.T) {
	head := http.Header{}
	setAssetHeaders(head, file.NewGetMediaType(contenttype.Open()).Execute("diagram.svg"))
	if got := head.Get("Content-Security-Policy"); !strings.Contains(got, "sandbox") {
		t.Fatalf("policy %q does not sandbox the picture", got)
	}
}
