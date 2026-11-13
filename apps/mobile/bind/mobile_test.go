package bind_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/apps/mobile/bind"
)

// graphCenter is the note of writeGraph with every relation: a parent, two
// children, a sibling, a jump and a reference.
const graphCenter = "Physics.md"

func writeGraph(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	notes := map[string]string{
		"Sciences.md": "---\ntitle: Sciences\nlinks:\n" +
			"  - to: \"[[Physics]]\"\n    role: child\n" +
			"  - to: \"[[Optics]]\"\n    role: child\n---\n\n# Sciences\n",
		"Physics.md": "---\ntitle: Physics\nlinks:\n" +
			"  - to: \"[[Sciences]]\"\n    role: parent\n" +
			"  - to: \"[[Entropy]]\"\n    role: child\n" +
			"  - to: \"[[Tides]]\"\n    role: child\n" +
			"  - to: \"[[Vellum]]\"\n    role: jump\n---\n\n# Physics\n\nSee [[Aurora]].\n",
		"Optics.md":  "# Optics\n",
		"Entropy.md": "# Entropy\n",
		"Tides.md":   "# Tides\n",
		"Vellum.md":  "# Vellum\n",
		"Aurora.md":  "# Aurora\n",
	}
	for name, text := range notes {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestStartAnswers is the whole of what a phone does: the core comes up, the
// listing arrives, a note is made, and the listing has it.
func TestStartAnswers(t *testing.T) {
	core := bind.NewCore()
	dir := t.TempDir()
	writeGraph(t, filepath.Join(dir, "vault"))
	port, err := core.Start(dir)
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() {
		if err := core.Stop(); err != nil {
			t.Error(err)
		}
	})

	ask := func(service, method string, body any) map[string]any {
		t.Helper()
		text, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("asking %s: %v", method, err)
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/numen.v1.%s/%s", port, service, method)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader(text))
		if err != nil {
			t.Fatalf("asking %s: %v", method, err)
		}
		req.Header.Set("Authorization", "Bearer "+core.Token())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("asking %s: %v", method, err)
		}
		defer func() {
			if err := res.Body.Close(); err != nil {
				t.Error(err)
			}
		}()
		response, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s answered %s: %s", method, res.Status, response)
		}
		var target map[string]any
		if err := json.Unmarshal(response, &target); err != nil {
			t.Fatalf("%s answered %q: %v", method, response, err)
		}
		return target
	}

	names := func() []string {
		var targetVal []string
		for _, entry := range asList(t, ask("FileService", "ListFiles", map[string]any{"folder": ""})["entries"]) {
			targetVal = append(targetVal, asString(t, asMap(t, entry)["name"]))
		}
		return targetVal
	}

	if got := len(names()); got == 0 {
		t.Fatal("the vault lists nothing")
	}

	// The vault is read behind the caller, so the relations are polled for.
	wanted := []string{"RELATION_PARENT", "RELATION_CHILD", "RELATION_JUMP", "RELATION_SIBLING", "RELATION_REFERENCE"}
	relations := map[string]int{}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	for ctx.Err() == nil {
		relations = map[string]int{}
		var around []any
		if related, hasRelated := ask("NoteService", "ListRelatedNotes", map[string]any{"path": graphCenter})["related"]; hasRelated && related != nil {
			around = asList(t, related)
		}
		for _, one := range around {
			relations[asString(t, asMap(t, one)["relation"])]++
		}
		if len(relations) == len(wanted) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	for _, relation := range wanted {
		if relations[relation] == 0 {
			t.Errorf("%s has no %s: %v", graphCenter, relation, relations)
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

// The three services this mounts read and write the user's notes, and the
// socket they listen on is reachable by every process on the phone. The only
// allowed origin is the page the platform serves; a request from any other
// origin receives no CORS grant.
// page is the one allowed origin, spelled out here and not read from the code
// under test.
const page = "http://localhost"

// The origin is set in the platform's own configuration and named here in
// `page`. The test compares the two.
func TestThePageIsServedFromTheOriginTheCoreAnswers(t *testing.T) {
	target, err := os.ReadFile(filepath.Join("..", "capacitor.config.ts"))
	if err != nil {
		t.Fatal(err)
	}
	scheme, _, found := strings.Cut(page, "://")
	if !found {
		t.Fatalf("%q names no scheme", page)
	}
	if want := "androidScheme: '" + scheme + "'"; !strings.Contains(string(target), want) {
		t.Errorf("the core answers %q and capacitor.config.ts says no %s", page, want)
	}
}

func TestTheSocketAnswersThePagesOwnOriginAndNoOther(t *testing.T) {
	core := bind.NewCore()
	if bind.Page != page {
		t.Fatalf("the page is served from %q and this reads %q", bind.Page, page)
	}
	port, err := core.Start(t.TempDir())
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() {
		if err := core.Stop(); err != nil {
			t.Error(err)
		}
	})

	url := fmt.Sprintf("http://127.0.0.1:%d/numen.v1.NoteService/CreateNote", port)
	for _, origin := range []string{page, "https://a-page-somebody-opened.example"} {
		for _, method := range []string{http.MethodOptions, http.MethodPost} {
			req, err := http.NewRequestWithContext(t.Context(), method, url, bytes.NewReader([]byte(`{"title":"Anemone"}`)))
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
			if err := res.Body.Close(); err != nil {
				t.Fatal(err)
			}
			if got := res.Header.Get("Access-Control-Allow-Origin"); got != page {
				t.Errorf("a %s from %s is allowed %q", method, origin, got)
			}
		}
	}
}

// The files of the vault and the data derived from them are not served. Every
// process on the phone can reach this socket, so a program acting for itself
// must not be able to read files off the disk or run a model through it.
func TestTheFilesOfTheVaultAreNotServedToThePhone(t *testing.T) {
	core := bind.NewCore()
	port, err := core.Start(t.TempDir())
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() {
		if err := core.Stop(); err != nil {
			t.Error(err)
		}
	})

	for _, item := range []string{
		"/assets/Physics.md",
		"/md.numen.readers.pdf/GetDocument",
		"/numen.v1.MediaService/ResolveAddress",
		"/numen.v1.TextService/ReadLiveText",
		"/numen.v1.TextService/CreateTextEdit",
	} {
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, item)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader([]byte(`{"path":"Physics.md"}`)))
		if err != nil {
			t.Fatalf("asking %s: %v", item, err)
		}
		req.Header.Set("Authorization", "Bearer "+core.Token())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("asking %s: %v", item, err)
		}
		if err := res.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s answered %s", item, res.Status)
		}
	}
}

// The vault, the index built from it and the settings file all reside in the
// directory passed to Start, so the platform's own backup is a path off the
// phone for the file the socket above does not serve. Backup is disabled, and
// nothing else in this repository reads the manifest.
func TestThePlatformDoesNotCarryTheVaultOffThePhone(t *testing.T) {
	pos := filepath.Join("..", "android", "app", "src", "main")
	manifest, err := os.ReadFile(filepath.Join(pos, "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, response := range []string{
		`android:allowBackup="false"`,
		`android:dataExtractionRules="@xml/data_extraction_rules"`,
	} {
		if !strings.Contains(string(manifest), response) {
			t.Errorf("the manifest does not say %s", response)
		}
	}

	rules, err := os.ReadFile(filepath.Join(pos, "res", "xml", "data_extraction_rules.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, responseVal := range []string{"<cloud-backup>", "<device-transfer>"} {
		if !strings.Contains(string(rules), responseVal) {
			t.Errorf("the rules name no %s", responseVal)
		}
	}
	if strings.Contains(string(rules), "<include") {
		t.Error("the rules take something off the phone and this test does not say what")
	}
}

// The installation's configuration file holds the model API keys.
// This build mounts no service for it, so nothing here reads that file out to a
// caller or overwrites it.
func TestTheSettingsFileIsNotServedToThePhone(t *testing.T) {
	core := bind.NewCore()
	dir := t.TempDir()
	secret := "sk-the-persons-own"
	content := fmt.Sprintf(`{"indexing":{"proofreading":{"profiles":{
		"openai": {"use":"service","name":"a-model","key":%q}
	}}}}`, secret)
	if err := os.WriteFile(filepath.Join(dir, "numen.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("writing the settings: %v", err)
	}

	port, err := core.Start(dir)
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() {
		if err := core.Stop(); err != nil {
			t.Error(err)
		}
	})

	for _, item := range []string{
		"/numen.v1.SettingsService/GetSettings",
		"/numen.v1.SettingsService/WriteSettings",
	} {
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, item)
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, bytes.NewReader([]byte(`{"written":"stolen: true\n"}`)))
		if err != nil {
			t.Fatalf("asking %s: %v", item, err)
		}
		req.Header.Set("Authorization", "Bearer "+core.Token())
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("asking %s: %v", item, err)
		}
		response, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := res.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if res.StatusCode == http.StatusOK {
			t.Errorf("%s answered %s: %s", item, res.Status, response)
		}
		if bytes.Contains(response, []byte(secret)) {
			t.Errorf("%s gave the key away: %s", item, response)
		}
	}

	target, err := os.ReadFile(filepath.Join(dir, "numen.json"))
	if err != nil {
		t.Fatalf("reading the settings back: %v", err)
	}
	if !bytes.Contains(target, []byte(secret)) {
		t.Errorf("the settings file was written over: %s", target)
	}
}

func asList(t *testing.T, value any) []any {
	t.Helper()
	list, isList := value.([]any)
	if !isList {
		t.Fatalf("%v is %T, want a JSON array", value, value)
	}
	return list
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()
	object, isObject := value.(map[string]any)
	if !isObject {
		t.Fatalf("%v is %T, want a JSON object", value, value)
	}
	return object
}

func asString(t *testing.T, value any) string {
	t.Helper()
	text, isString := value.(string)
	if !isString {
		t.Fatalf("%v is %T, want a JSON string", value, value)
	}
	return text
}

// Every process on the phone can connect to the loopback port, so a request
// without the token of this start is refused and one with it is answered.
func TestTheServerAnswersOnlyARequestWithTheToken(t *testing.T) {
	core := bind.NewCore()
	port, err := core.Start(t.TempDir())
	if err != nil {
		t.Fatalf("starting: %v", err)
	}
	t.Cleanup(func() {
		if err := core.Stop(); err != nil {
			t.Error(err)
		}
	})
	if core.Token() == "" {
		t.Fatal("a running core has no token")
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/numen.v1.FileService/ListFiles", port)
	for name, authorization := range map[string]string{
		"none":          "",
		"another token": "Bearer " + strings.Repeat("0", len(core.Token())),
		"no scheme":     core.Token(),
		"the token":     "Bearer " + core.Token(),
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(`{"folder":""}`))
		if err != nil {
			t.Fatalf("asking with %s: %v", name, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("asking with %s: %v", name, err)
		}
		if err := res.Body.Close(); err != nil {
			t.Fatal(err)
		}
		want := http.StatusUnauthorized
		if name == "the token" {
			want = http.StatusOK
		}
		if res.StatusCode != want {
			t.Errorf("a request with %s answered %s, want %d", name, res.Status, want)
		}
	}
}

func TestTheTokenChangesWithEveryStart(t *testing.T) {
	core := bind.NewCore()
	if core.Token() != "" {
		t.Fatal("a core that is not running has a token")
	}
	var tokens []string
	for range 2 {
		if _, err := core.Start(t.TempDir()); err != nil {
			t.Fatalf("starting: %v", err)
		}
		tokens = append(tokens, core.Token())
		if err := core.Stop(); err != nil {
			t.Fatal(err)
		}
	}
	if tokens[0] == "" || tokens[0] == tokens[1] {
		t.Errorf("tokens %q", tokens)
	}
}
