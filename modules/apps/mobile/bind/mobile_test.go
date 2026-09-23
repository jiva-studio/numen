package bind_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		for _, entry := range ask("FileService", "ListFiles", map[string]any{"folder": ""})["entries"].([]any) {
			held = append(held, entry.(map[string]any)["name"].(string))
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
		around, _ := ask("NoteService", "GetNeighbourhood",
			map[string]any{"path": bind.Seeded})["related"].([]any)
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

	made := ask("NoteService", "CreateNote", map[string]any{"title": "Anemone", "folder": ""})
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

// The three services this mounts read and write the person's notes, and the
// socket they stand on is one every process on the phone reaches. So the one
// origin it answers is the page the platform serves: a page in the person's own
// browser asks the same address and is refused by the browser before the ask
// leaves it.
// page is the one origin, spelled out here and read from nothing under test.
const page = "http://localhost"

// The origin is settled in the platform's own configuration and named here in
// Go, and a scheme changed in one is a page refused by the other with nothing
// said. The two are read against each other.
func TestThePageIsServedFromTheOriginTheCoreAnswers(t *testing.T) {
	held, err := os.ReadFile(filepath.Join("..", "capacitor.config.ts"))
	if err != nil {
		t.Fatal(err)
	}
	scheme, _, found := strings.Cut(page, "://")
	if !found {
		t.Fatalf("%q names no scheme", page)
	}
	if want := "androidScheme: '" + scheme + "'"; !strings.Contains(string(held), want) {
		t.Errorf("the core answers %q and capacitor.config.ts says no %s", page, want)
	}
}

func TestTheSocketAnswersThePagesOwnOriginAndNoOther(t *testing.T) {
	if bind.Page != page {
		t.Fatalf("the page is served from %q and this reads %q", bind.Page, page)
	}
	port, err := bind.Start(t.TempDir())
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = bind.Stop() })

	url := fmt.Sprintf("http://127.0.0.1:%d/numen.v1.NoteService/CreateNote", port)
	for _, origin := range []string{page, "https://a-page-somebody-opened.example"} {
		for _, method := range []string{http.MethodOptions, http.MethodPost} {
			req, err := http.NewRequest(method, url, bytes.NewReader([]byte(`{"title":"Anemone"}`)))
			if err != nil {
				t.Fatalf("asking as %s: %v", origin, err)
			}
			req.Header.Set("Origin", origin)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Connect-Protocol-Version", "1")
			if method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("asking as %s: %v", origin, err)
			}
			res.Body.Close()
			if got := res.Header.Get("Access-Control-Allow-Origin"); got != page {
				t.Errorf("a %s from %s is allowed %q", method, origin, got)
			}
		}
	}
}

// The files of the vault and what is made from them are not served here. Every
// process on the phone reaches this socket and a program speaking for itself is
// held to no origin, so a caller that reached it must not be able to read a book
// off the disk or set a model running over one.
func TestTheFilesOfTheVaultAreNotServedToThePhone(t *testing.T) {
	port, err := bind.Start(t.TempDir())
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() { _ = bind.Stop() })

	for _, at := range []string{
		"/assets/Physics.md",
		"/numen.v1.DocumentService/GetDocument",
		"/numen.v1.RecordingService/GetRecording",
		"/numen.v1.OcrService/ReadOcr",
		"/numen.v1.TranscriptService/ReadTranscript",
		"/numen.v1.TranscriptService/WriteTranscript",
		"/numen.v1.ArticleService/ReadArticle",
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

// The vault, the index built from it and the settings file all sit under the
// directory Start is handed, so the platform's own backup is a way off the
// phone for the file the socket above refuses to serve. It is turned off, and
// nothing else in this repository reads the manifest.
func TestThePlatformDoesNotCarryTheVaultOffThePhone(t *testing.T) {
	at := filepath.Join("..", "android", "app", "src", "main")
	manifest, err := os.ReadFile(filepath.Join(at, "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, said := range []string{
		`android:allowBackup="false"`,
		`android:dataExtractionRules="@xml/data_extraction_rules"`,
	} {
		if !strings.Contains(string(manifest), said) {
			t.Errorf("the manifest does not say %s", said)
		}
	}

	rules, err := os.ReadFile(filepath.Join(at, "res", "xml", "data_extraction_rules.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, said := range []string{"<cloud-backup>", "<device-transfer>"} {
		if !strings.Contains(string(rules), said) {
			t.Errorf("the rules name no %s", said)
		}
	}
	if strings.Contains(string(rules), "<include") {
		t.Error("the rules take something off the phone and this test does not say what")
	}
}

// The file a person configures the installation in holds the keys it reaches
// models with. This build mounts no service about it, so nothing here reads that
// file out to a caller or writes another one in its place.
func TestTheSettingsFileIsNotServedToThePhone(t *testing.T) {
	dir := t.TempDir()
	secret := "sk-the-persons-own"
	written := fmt.Sprintf(`{"indexing":{"proofreading":{"profiles":{
		"openai": {"use":"service","name":"a-model","key":%q}
	}}}}`, secret)
	if err := os.WriteFile(filepath.Join(dir, "numen.json"), []byte(written), 0o600); err != nil {
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

	held, err := os.ReadFile(filepath.Join(dir, "numen.json"))
	if err != nil {
		t.Fatalf("reading the settings back: %v", err)
	}
	if !bytes.Contains(held, []byte(secret)) {
		t.Errorf("the settings file was written over: %s", held)
	}
}
