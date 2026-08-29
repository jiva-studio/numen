package bind_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

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

	ask := func(method string, body any) map[string]any {
		t.Helper()
		text, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("asking %s: %v", method, err)
		}
		url := fmt.Sprintf("http://127.0.0.1:%d/numen.v1.VaultService/%s", port, method)
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
		for _, entry := range ask("List", map[string]any{"folder": ""})["entries"].([]any) {
			held = append(held, entry.(map[string]any)["name"].(string))
		}
		return held
	}

	if got := len(names()); got != 3 {
		t.Fatalf("the vault it seeded lists %d entries, want 3: %v", got, names())
	}

	made := ask("Create", map[string]any{"title": "Anemone", "folder": ""})
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
