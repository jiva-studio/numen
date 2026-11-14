package service

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/api"
)

const meeting = "---\ntype: meeting\n---\n# {{title}}\n\nAttendees\n"

func getToday() time.Time { return time.Date(2026, 11, 14, 9, 5, 0, 0, time.UTC) }

func newWeb(t *testing.T, vault *hosttest.Vault) *httptest.Server {
	t.Helper()
	source := new(clock.Fake)
	source.Set(getToday())
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		return mount(s, vault, source, ulid.NewGenerator(rand.Reader))
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return web
}

func post(t *testing.T, web *httptest.Server, path, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer plugin-token")
	res, err := web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, data
}

func createNote(t *testing.T, web *httptest.Server, template, folder string) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(api.CreateNoteRequest{Template: template, Folder: folder})
	if err != nil {
		t.Fatal(err)
	}
	return post(t, web, "/"+api.Service+"/"+api.MethodCreateNote, string(body))
}

func readFailure(t *testing.T, body []byte) server.APIError {
	t.Helper()
	var failure server.APIError
	if err := json.Unmarshal(body, &failure); err != nil {
		t.Fatalf("%s: %v", body, err)
	}
	return failure
}

func TestCreateNoteReturnsThePathOfTheNewNote(t *testing.T) {
	vault := hosttest.New("notes")
	vault.File("Templates/Meeting.md", meeting)
	web := newWeb(t, vault)

	status, body := createNote(t, web, "Templates/Meeting.md", "work")

	var got api.CreateNoteResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || got.Path != "work/Meeting.md" {
		t.Fatalf("CreateNote = %d %s", status, body)
	}
	if content, isThere := vault.Content("work/Meeting.md"); !isThere || !strings.Contains(content, "# Meeting\n") {
		t.Errorf("the note is %q (%v)", content, isThere)
	}
}

func TestMapTheErrorsOfTheUseCaseToTheirCodes(t *testing.T) {
	t.Run("maps the errors of the use case to their codes", func(t *testing.T) {
		manyNames := hosttest.New("notes")
		manyNames.File("Templates/Meeting.md", meeting)
		manyNames.File("work/Meeting.md", "taken")
		for number := 2; number <= 100; number++ {
			manyNames.File(fmt.Sprintf("work/Meeting %d.md", number), "taken")
		}
		noVault := hosttest.New("notes")
		noVault.File("Templates/Meeting.md", meeting)
		noVault.Hide()
		denied := hosttest.New("notes")
		denied.File("Templates/Meeting.md", meeting)
		denied.Refuse("Templates/Meeting.md", host.ErrPermissionDenied)
		invalid := hosttest.New("notes")
		invalid.File("Templates/Plan: draft.md", "---\nname: {{title}}\n---\n")
		binary := hosttest.New("notes")
		binary.File("Templates/Binary.md", "\xff\xfe\x00")
		plain := hosttest.New("notes")
		plain.File("Templates/Meeting.md", meeting)
		plain.File("Notes/Meeting.md", meeting)

		for _, c := range []struct {
			name     string
			vault    *hosttest.Vault
			template string
			status   int
			code     server.ErrorCode
			message  string
		}{
			{"a path outside the folder", plain, "Notes/Meeting.md", 400, server.CodeInvalidArgument, "not a template: Notes/Meeting.md"},
			{"a parent segment", plain, "Templates/../Notes/Meeting.md", 400, server.CodeInvalidArgument, "not a template: Templates/../Notes/Meeting.md"},
			{"a path that is not a note", plain, "Templates/Meeting.txt", 400, server.CodeInvalidArgument, "not a template: Templates/Meeting.txt"},
			{"a missing template", plain, "Templates/Nothing.md", 404, server.CodeNotFound, "template not found: Templates/Nothing.md"},
			{"frontmatter that is invalid after substitution", invalid, "Templates/Plan: draft.md", 412, server.CodeFailedPrecondition, "template does not produce a valid note: "},
			{"a template that is not text", binary, "Templates/Binary.md", 412, server.CodeFailedPrecondition, "template does not produce a valid note: "},
			{"no free name", manyNames, "Templates/Meeting.md", 412, server.CodeFailedPrecondition, "no free name for Meeting"},
			{"no vault", noVault, "Templates/Meeting.md", 412, server.CodeFailedPrecondition, ""},
			{"a denied host call", denied, "Templates/Meeting.md", 403, server.CodePermissionDenied, ""},
		} {
			t.Run(c.name, func(t *testing.T) {
				status, body := createNote(t, newWeb(t, c.vault), c.template, "work")

				failure := readFailure(t, body)
				if status != c.status || failure.Code != c.code || !strings.HasPrefix(failure.Message, c.message) {
					t.Errorf("CreateNote = %d %s, want %d %s %q", status, body, c.status, c.code, c.message)
				}
			})
		}
	})
}

func TestTheHostConfiguresTheFolderOfTheTemplates(t *testing.T) {
	vault := hosttest.New("notes")
	vault.File("Templates/Meeting.md", meeting)
	vault.File("Models/Book.md", meeting)
	web := newWeb(t, vault)

	if status, body := createNote(t, web, "Models/Book.md", "work"); status != http.StatusBadRequest {
		t.Errorf("before the setting: CreateNote = %d %s", status, body)
	}
	if status, body := post(t, web, "/numen.v1.PluginLifecycleService/Configure", `{"values":{"folder":" /Models/ "}}`); status != http.StatusOK {
		t.Fatalf("Configure = %d %s", status, body)
	}

	if status, body := createNote(t, web, "Models/Book.md", "work"); status != http.StatusOK {
		t.Errorf("after the setting: CreateNote = %d %s", status, body)
	}
	if status, body := createNote(t, web, "Templates/Meeting.md", "work"); status != http.StatusBadRequest {
		t.Errorf("the old folder: CreateNote = %d %s", status, body)
	}
}

func TestNormalizeTheFolderSetting(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/folder.json")
	if err != nil {
		t.Fatal(err)
	}
	var table map[string][]struct {
		Raw    string `json:"raw"`
		Folder string `json:"folder"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"reads an empty folder as Templates", "drops slashes and dot segments", "reads a parent segment as Templates"} {
		t.Run(name, func(t *testing.T) {
			if len(table[name]) == 0 {
				t.Fatal("no cases in testdata/folder.json")
			}
			for _, c := range table[name] {
				if got := normalizeFolder(c.Raw); got != c.Folder {
					t.Errorf("normalizeFolder(%q) = %q, want %q", c.Raw, got, c.Folder)
				}
			}
		})
	}
}
