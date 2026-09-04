package bind_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/mobile/bind"
)

// TestStartAnswers is the whole of what a phone does: the core comes up, the
// listing arrives, a note is made, and the listing has it.
func TestStartAnswers(t *testing.T) {
	dir := t.TempDir()
	port, err := bind.Start(dir)
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = bind.Stop() })

	ask := func(service, method string, body any) map[string]any {
		t.Helper()
		text, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("asking %s: %v", method, err)
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/numen.v1.%s/%s", port, service, method)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(text))
		if err != nil {
			t.Fatalf("asking %s: %v", method, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("asking %s: %v", method, err)
		}
		defer res.Body.Close()
		said, _ := io.ReadAll(res.Body)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s answered %s: %s", method, res.Status, said)
		}
		var held map[string]any
		if err := json.Unmarshal(said, &held); err != nil {
			t.Fatalf("%s answered %q: %v", method, said, err)
		}
		return held
	}

	names := func() []string {
		var held []string
		for _, entry := range ask("FileService", "List", map[string]any{"folder": ""})["entries"].([]any) {
			held = append(held, entry.(map[string]any)["displayName"].(string))
		}
		return held
	}

	if got := len(names()); got == 0 {
		t.Fatal("the vault it seeded lists nothing")
	}

	// The note it opens on stands in every seat at once, which is what makes a
	// picture worth drawing. The vault is read behind the caller, so the seats
	// arrive rather than being there.
	wanted := []string{"SEAT_PARENT", "SEAT_CHILD", "SEAT_JUMP", "SEAT_SIBLING"}
	seats := map[string]int{}
	for until := time.Now().Add(10 * time.Second); time.Now().Before(until); {
		seats = map[string]int{}
		around, _ := ask("NoteService", "Neighbourhood", map[string]any{"path": bind.Seeded})["related"].([]any)
		for _, one := range around {
			seats[one.(map[string]any)["seat"].(string)]++
		}
		if len(seats) == len(wanted) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, seat := range wanted {
		if seats[seat] == 0 {
			t.Errorf("%s has no %s: %v", bind.Seeded, seat, seats)
		}
	}

	made := ask("NoteService", "Create", map[string]any{"title": "Anemone", "folder": ""})
	if made["path"] != "Anemone.md" {
		t.Fatalf("made %v, want Anemone.md", made)
	}

	found := false
	for _, name := range names() {
		if name == "Anemone.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the note it made is not in the listing: %v", names())
	}
}

// The files of the vault and what is made from them are not served here. What
// this server answers is answered to any origin at all, so a caller that
// reached it must not be able to read a book off the disk or set a model
// running over one.
func TestTheFilesOfTheVaultAreNotServedToThePhone(t *testing.T) {
	port, err := bind.Start(t.TempDir())
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = bind.Stop() })

	for _, at := range []string{
		"/assets/Physics.md",
		"/numen.v1.AssetService/GetDocument",
		"/numen.v1.AssetService/GetRecording",
		"/numen.v1.AssetService/ListHighlights",
		"/numen.v1.ArtifactService/ListArtifacts",
		"/numen.v1.ArtifactService/CreateArtifact",
		"/numen.v1.ArtifactService/DeleteArtifact",
	} {
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, at)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(`{"path":"Physics.md"}`)))
		if err != nil {
			t.Fatalf("asking %s: %v", at, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("asking %s: %v", at, err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s answered %s", at, res.Status)
		}
	}
}

// The file a person configures the installation in holds the keys it reaches
// models with. This build binds no setting, so nothing here reads that file out
// to a caller or writes another one in its place.
func TestTheSettingsFileIsNotServedToThePhone(t *testing.T) {
	dir := t.TempDir()
	secret := "sk-the-persons-own"
	written := fmt.Sprintf(`{"indexing":{"proofreading":{"profiles":{
		"openai": {"use":"service","name":"a-model","key":%q}
	}}}}`, secret)
	if err := os.WriteFile(filepath.Join(dir, "settings.yaml"), []byte(written), 0o600); err != nil {
		t.Fatalf("writing the settings: %v", err)
	}

	port, err := bind.Start(dir)
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = bind.Stop() })

	for _, at := range []string{
		"/numen.v1.SettingsService/GetSettings",
		"/numen.v1.SettingsService/ReadSettingsFile",
		"/numen.v1.SettingsService/WriteSettings",
		"/numen.v1.SettingsService/WriteSettingsFile",
	} {
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, at)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(`{"written":"stolen: true\n"}`)))
		if err != nil {
			t.Fatalf("asking %s: %v", at, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("asking %s: %v", at, err)
		}
		said, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode == http.StatusOK {
			t.Errorf("%s answered %s: %s", at, res.Status, said)
		}
		if bytes.Contains(said, []byte(secret)) {
			t.Errorf("%s gave the key away: %s", at, said)
		}
	}

	held, err := os.ReadFile(filepath.Join(dir, "settings.yaml"))
	if err != nil {
		t.Fatalf("reading the settings back: %v", err)
	}
	if !bytes.Contains(held, []byte(secret)) {
		t.Errorf("the settings file was written over: %s", held)
	}
}
