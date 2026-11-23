package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	pathpkg "path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/mask"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/usecase/newnote"
)

const meeting = "---\ntype: meeting\n---\n# {{ .title }}\n\nAttendees\n"

func getToday() time.Time { return time.Date(2026, 11, 14, 9, 5, 0, 0, time.UTC) }

type testWeb struct {
	*httptest.Server
	vault *hosttest.Vault
}

func newWeb(t *testing.T, vault *hosttest.Vault) *testWeb {
	t.Helper()
	source := new(clocktest.Fake)
	source.Set(getToday())
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		return mount(s, vault, source, ulid.NewGenerator(rand.Reader))
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return &testWeb{Server: web, vault: vault}
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

func createNote(t *testing.T, web *testWeb, template, folder string) (int, []byte) {
	t.Helper()
	return createNoteWith(t, web, template, folder, nil)
}

func createNoteAs(t *testing.T, web *testWeb, template, folder, name string) (int, []byte) {
	t.Helper()
	content, _ := web.vault.ReadContent(template)
	body, err := json.Marshal(api.CreateNoteRequest{Template: template, Folder: folder, Name: name, Values: nil, Etag: newnote.Etag(content)})
	if err != nil {
		t.Fatal(err)
	}
	return post(t, web.Server, "/"+api.Service+"/"+api.MethodCreateNote, string(body))
}

func listFields(t *testing.T, web *testWeb, template string) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(api.ListFieldsRequest{Template: template, Folder: "work"})
	if err != nil {
		t.Fatal(err)
	}
	return post(t, web.Server, "/"+api.Service+"/"+api.MethodListFields, string(body))
}

func createNoteWith(t *testing.T, web *testWeb, template, folder string, values map[string]string) (int, []byte) {
	t.Helper()
	content, _ := web.vault.ReadContent(template)
	name := strings.TrimSuffix(pathpkg.Base(template), ".md")
	body, err := json.Marshal(api.CreateNoteRequest{Template: template, Folder: folder, Name: name, Values: values, Etag: newnote.Etag(content)})
	if err != nil {
		t.Fatal(err)
	}
	return post(t, web.Server, "/"+api.Service+"/"+api.MethodCreateNote, string(body))
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
	vault.WriteFile("Templates/Meeting.md", meeting)
	web := newWeb(t, vault)

	status, body := createNote(t, web, "Templates/Meeting.md", "work")

	var got api.CreateNoteResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || got.Path != "work/Meeting.md" || got.Warning != "" {
		t.Fatalf("CreateNote = %d %s", status, body)
	}
	if content, isThere := vault.ReadContent("work/Meeting.md"); !isThere || !strings.Contains(content, "# Meeting\n") {
		t.Errorf("the note is %q (%v)", content, isThere)
	}
}

const asking = "# {{ prompt \"Client\" \"Acme\" }} {{ select \"Kind\" \"call\" \"visit\" }}\n"

func TestListFieldsReturnsWhatTheTemplateAsks(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Visit.md", asking)
	vault.WriteFile("Templates/Plain.md", meeting)
	web := newWeb(t, vault)

	status, body := listFields(t, web, "Templates/Visit.md")
	var got api.ListFieldsResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	want := []api.Field{
		{Kind: api.KindText, Label: "Client", Options: []string{}, Default: "Acme"},
		{Kind: api.KindSelect, Label: "Kind", Options: []string{"call", "visit"}, Default: "call"},
	}
	if status != http.StatusOK || !reflect.DeepEqual(got.Fields, want) || got.Name != "Visit" {
		t.Errorf("ListFields = %d %s", status, body)
	}
	vault.WriteFile("work/Plain.md", "taken")
	if status, body := listFields(t, web, "Templates/Plain.md"); status != http.StatusOK || !strings.Contains(string(body), `"fields":[]`) || !strings.Contains(string(body), `"name":"Plain 2"`) {
		t.Errorf("ListFields of a plain template = %d %s", status, body)
	}
	if status, body := listFields(t, web, "Notes/x.md"); status != http.StatusPreconditionFailed {
		t.Errorf("ListFields of a note that is no template = %d %s", status, body)
	}
}

func TestCreateNoteRefusesATemplateThatChanged(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Visit.md", asking)
	web := newWeb(t, vault)
	_, listed := listFields(t, web, "Templates/Visit.md")
	var fields api.ListFieldsResponse
	if err := json.Unmarshal(listed, &fields); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(api.CreateNoteRequest{Template: "Templates/Visit.md", Folder: "work", Name: fields.Name, Values: nil, Etag: fields.Etag})
	if err != nil {
		t.Fatal(err)
	}
	path := "/" + api.Service + "/" + api.MethodCreateNote
	vault.WriteFile("Templates/Visit.md", asking+"changed\n")

	status, answer := post(t, web.Server, path, string(body))

	failure := readFailure(t, answer)
	if status != http.StatusPreconditionFailed || failure.Code != server.CodeFailedPrecondition || failure.Message != "template changed, open the picker again" {
		t.Errorf("CreateNote of a changed template = %d %s", status, answer)
	}
	if _, isThere := vault.ReadContent("work/Visit.md"); isThere {
		t.Error("a note was made")
	}
}

func TestCreateNoteWritesTheValues(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Visit.md", asking)
	web := newWeb(t, vault)

	status, body := createNoteWith(t, web, "Templates/Visit.md", "work", map[string]string{"Client": "Globex", "Kind": "visit"})

	if content, _ := vault.ReadContent("work/Visit.md"); status != http.StatusOK || !strings.Contains(content, "# Globex visit\n") {
		t.Errorf("CreateNote = %d %s, note %q", status, body, content)
	}
	vault.WriteFile("Templates/Other.md", asking)
	status, body = createNoteWith(t, web, "Templates/Other.md", "work", map[string]string{"Kind": "nap"})
	if failure := readFailure(t, body); status != http.StatusPreconditionFailed || failure.Code != server.CodeFailedPrecondition || !strings.Contains(failure.Message, "Kind") {
		t.Errorf("CreateNote with a bad option = %d %s", status, body)
	}
}

func TestCreateNoteRefusesAFolderOutsideTheVault(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Meeting.md", meeting)
	web := newWeb(t, vault)

	status, body := createNote(t, web, "Templates/Meeting.md", "../x")

	failure := readFailure(t, body)
	if status != http.StatusPreconditionFailed || failure.Code != server.CodeFailedPrecondition || !strings.Contains(failure.Message, "not a folder of the vault") {
		t.Errorf("CreateNote = %d %s", status, body)
	}
}

func TestMapTheErrorsOfTheUseCaseToTheirCodes(t *testing.T) {
	t.Run("maps the errors of the use case to their codes", func(t *testing.T) {
		taken := hosttest.New("notes")
		taken.WriteFile("Templates/Meeting.md", meeting)
		taken.WriteFile("work/Meeting.md", "taken")
		broken := hosttest.New("notes")
		broken.WriteFile("Templates/Broken.md", "# {{ .title }}\n{{ if }}\n")
		noVault := hosttest.New("notes")
		noVault.WriteFile("Templates/Meeting.md", meeting)
		noVault.Hide()
		denied := hosttest.New("notes")
		denied.WriteFile("Templates/Meeting.md", meeting)
		denied.Refuse("Templates/Meeting.md", host.ErrPermissionDenied)
		invalid := hosttest.New("notes")
		invalid.WriteFile("Templates/@draft.md", "---\nname: {{ .title }}\n---\n")
		binary := hosttest.New("notes")
		binary.WriteFile("Templates/Binary.md", "\xff\xfe\x00")
		plain := hosttest.New("notes")
		plain.WriteFile("Templates/Meeting.md", meeting)
		plain.WriteFile("Notes/Meeting.md", meeting)

		for _, c := range []struct {
			name     string
			vault    *hosttest.Vault
			template string
			status   int
			code     server.ErrorCode
			message  string
		}{
			{"a path no mask names", plain, "Notes/Meeting.md", 412, server.CodeFailedPrecondition, "not a template: Notes/Meeting.md"},
			{"a parent segment", plain, "Templates/../Notes/Meeting.md", 412, server.CodeFailedPrecondition, "not a template: Templates/../Notes/Meeting.md"},
			{"a path that is not a note", plain, "Templates/Meeting.txt", 412, server.CodeFailedPrecondition, "not a template: Templates/Meeting.txt"},
			{"a missing template", plain, "Templates/Nothing.md", 404, server.CodeNotFound, "template not found: Templates/Nothing.md"},
			{"frontmatter that is invalid after substitution", invalid, "Templates/@draft.md", 412, server.CodeFailedPrecondition, "template does not produce a valid note: "},
			{"a template that is not text", binary, "Templates/Binary.md", 412, server.CodeFailedPrecondition, "template does not produce a valid note: "},
			{"a template that does not parse", broken, "Templates/Broken.md", 412, server.CodeFailedPrecondition, "template does not produce a valid note: invalid template: template: Broken.md:2"},
			{"a name that is taken", taken, "Templates/Meeting.md", 409, server.CodeAlreadyExists, "a note of that name exists: work/Meeting.md"},
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

func TestCreateNoteRefusesANameNoNoteCanHave(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Meeting.md", meeting)
	web := newWeb(t, vault)

	for _, name := range []string{"", "a/b", "..", "a:b"} {
		status, body := createNoteAs(t, web, "Templates/Meeting.md", "work", name)

		failure := readFailure(t, body)
		if status != http.StatusBadRequest || failure.Code != server.CodeInvalidArgument || !strings.Contains(failure.Message, "cannot be given this title") {
			t.Errorf("CreateNote named %q = %d %s", name, status, body)
		}
	}
}

func TestCreateNoteUsesTheNameGiven(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Meeting.md", meeting)
	web := newWeb(t, vault)

	status, body := createNoteAs(t, web, "Templates/Meeting.md", "work", "Kickoff")

	if content, isThere := vault.ReadContent("work/Kickoff.md"); status != http.StatusOK || !isThere || !strings.Contains(content, "# Kickoff\n") {
		t.Errorf("CreateNote = %d %s, note %q", status, body, content)
	}
}

func TestTheHostConfiguresTheMasksOfTheTemplates(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Meeting.md", meeting)
	vault.WriteFile("Models/Book.md", meeting)
	web := newWeb(t, vault)

	if status, body := createNote(t, web, "Models/Book.md", "work"); status != http.StatusPreconditionFailed {
		t.Errorf("before the setting: CreateNote = %d %s", status, body)
	}
	if status, body := post(t, web.Server, "/numen.v1.PluginLifecycleService/Configure", `{"values":{"templates":" /Models/** , **/*.tpl.md "}}`); status != http.StatusOK {
		t.Fatalf("Configure = %d %s", status, body)
	}

	if status, body := createNote(t, web, "Models/Book.md", "work"); status != http.StatusOK {
		t.Errorf("after the setting: CreateNote = %d %s", status, body)
	}
	if status, body := createNote(t, web, "Templates/Meeting.md", "work"); status != http.StatusPreconditionFailed {
		t.Errorf("the old masks: CreateNote = %d %s", status, body)
	}
}

type unindexed struct{ *hosttest.Vault }

func (unindexed) Scan(context.Context, []string) error { return errors.New("index is down") }

func TestAnswerWithTheNoteWhenItCannotBeIndexed(t *testing.T) {
	vault := hosttest.New("notes")
	vault.WriteFile("Templates/Meeting.md", meeting)
	useCase := newnote.NewCreateFromTemplate(unindexed{vault}, new(clocktest.Fake), ulid.NewGenerator(rand.Reader), func() mask.Set { return mask.Parse("") })

	got, err := makeCreateNoteHandler(useCase)(t.Context(), api.CreateNoteRequest{Template: "Templates/Meeting.md", Folder: "work", Name: "Meeting", Values: nil, Etag: newnote.Etag(meeting)})

	if err != nil || got.Path != "work/Meeting.md" || !strings.Contains(got.Warning, "not indexed") || !strings.Contains(got.Warning, "index is down") {
		t.Errorf("CreateNote = %+v, %v", got, err)
	}
}
