package service_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/service"
	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

const (
	tagsServiceName = "md.numen.tools.taxonomy"
	tagsFixtureDir  = "../../testdata/api"
	tagsToken       = "plugin-token"
)

func newHarborNotes() map[string]string {
	return map[string]string{
		"notes/Quay.md": "---\ntags: [Port, Sea]\n---\nThe quay.\n",
		"notes/Dock.md": "---\ntags: [Port, Sea, Quay]\n---\nThe dock.\n",
		"notes/Pier.md": "The pier.\n",
	}
}

type taggingPlugin struct {
	vault *hosttest.Vault
	web   *httptest.Server
}

func newTaggingPlugin(t *testing.T, files map[string]string) taggingPlugin {
	t.Helper()
	v := hosttest.New("garden")
	for path, content := range files {
		v.WriteFile(path, content)
	}
	app, err := service.NewApp(func(id domain.VaultID) tagcache.Vault {
		return tagcache.Vault{Frontmatter: v.With(id), Feed: v.With(id), Store: v.Store().With(id), Properties: v.With(id)}
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := servertest.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": tagsToken}, func(s *server.Server) error {
		app.Mount(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return taggingPlugin{vault: v, web: web}
}

func (p taggingPlugin) call(t *testing.T, method string, request []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, p.web.URL+"/"+tagsServiceName+"/"+method, bytes.NewReader(request))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tagsToken)
	req.Header.Set("Numen-Vault", "garden")
	res, err := p.web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Log(err)
		}
	}()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, body
}

func readTagsFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(tagsFixtureDir, name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func isSameJSON(t *testing.T, got, want []byte) bool {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		return false
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(gotValue, wantValue)
}

func (p taggingPlugin) getTagList(t *testing.T) api.TagList {
	t.Helper()
	status, body := p.call(t, api.MethodGetTags, []byte(`{}`))
	var list api.TagList
	if status != http.StatusOK {
		t.Fatalf("GetTags = %d %s, want 200", status, body)
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func (p taggingPlugin) setTags(t *testing.T, path string, toggles ...string) api.SetTagsResult {
	t.Helper()
	request, err := json.Marshal(api.SetTagsRequest{Paths: []string{path}, Toggles: toggles})
	if err != nil {
		t.Fatal(err)
	}
	status, body := p.call(t, api.MethodSetTags, request)
	var result api.SetTagsResult
	if status != http.StatusOK {
		t.Fatalf("SetTags = %d %s, want 200", status, body)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func (p taggingPlugin) getNoteTags(t *testing.T, path string) []string {
	t.Helper()
	page, err := p.vault.ReadFrontmatter(t.Context(), host.FrontmatterQuery{Paths: []string{path}})
	if err != nil || len(page.Notes) != 1 {
		t.Fatalf("read %s: %d notes (%v)", path, len(page.Notes), err)
	}
	return tags.Parse(page.Notes[0].Values[tags.FrontmatterKey])
}

func listTagNames(list api.TagList) []string {
	var names []string
	for _, one := range list.Tags {
		names = append(names, one.Tag)
	}
	return names
}

func TestGetTagsAnswersEveryTagOfTheVaultWithItsCount(t *testing.T) {
	p := newTaggingPlugin(t, newHarborNotes())

	status, body := p.call(t, api.MethodGetTags, readTagsFixture(t, "GetTags.request.json"))

	if status != http.StatusOK || !isSameJSON(t, body, readTagsFixture(t, "GetTags.response.json")) {
		t.Errorf("GetTags = %d %s, want 200 and the fixture GetTags.response.json", status, body)
	}
}

func TestSetTagsWritesTheTagsToTheNoteAndAnswersWithTheTagsItHasNow(t *testing.T) {
	p := newTaggingPlugin(t, newHarborNotes())

	status, body := p.call(t, api.MethodSetTags, readTagsFixture(t, "SetTags.request.json"))

	if status != http.StatusOK || !isSameJSON(t, body, readTagsFixture(t, "SetTags.response.json")) {
		t.Errorf("SetTags = %d %s, want 200 and the fixture SetTags.response.json", status, body)
	}
	if got := p.getNoteTags(t, "notes/Pier.md"); !slices.Equal(got, []string{"Port", "Sea"}) {
		t.Errorf("the pier has the tags %q, want Port and Sea", got)
	}
	if content, _ := p.vault.ReadContent("notes/Pier.md"); !strings.Contains(content, "The pier.") {
		t.Errorf("the pier is %q, want its text kept", content)
	}
}

func TestSetTagsReportsAnErrorForANoteItCannotWrite(t *testing.T) {
	p := newTaggingPlugin(t, newHarborNotes())
	p.vault.RefuseWriting("notes/Pier.md", errors.New("disk full"))

	result := p.setTags(t, "notes/Pier.md", "Port")

	if len(result.Notes) != 1 || result.Notes[0].Path != "notes/Pier.md" || result.Notes[0].Error == "" {
		t.Errorf("result = %+v, want the pier with an error", result)
	}
}

func TestGetTagsListsATagOnceANoteHasItAndNotAfterItIsCleared(t *testing.T) {
	p := newTaggingPlugin(t, newHarborNotes())

	p.setTags(t, "notes/Pier.md", "Wharf")
	listed := p.getTagList(t)
	p.setTags(t, "notes/Pier.md", "Wharf")
	cleared := p.getTagList(t)

	if !slices.Contains(listTagNames(listed), "Wharf") {
		t.Errorf("tags = %q, want Wharf once a note has it", listTagNames(listed))
	}
	if slices.Contains(listTagNames(cleared), "Wharf") {
		t.Errorf("tags = %q, want no Wharf once no note has it", listTagNames(cleared))
	}
}

func TestGetTagsReturnsTheRecentTagsThatANoteStillHas(t *testing.T) {
	p := newTaggingPlugin(t, newHarborNotes())
	p.setTags(t, "notes/Pier.md", "Wharf")
	p.setTags(t, "notes/Pier.md", "Sea")

	set := p.getTagList(t)
	p.setTags(t, "notes/Pier.md", "Wharf")
	cleared := p.getTagList(t)

	if !slices.Equal(set.Recent, []string{"Sea", "Wharf"}) {
		t.Errorf("recent = %q, want Sea then Wharf", set.Recent)
	}
	if !slices.Equal(cleared.Recent, []string{"Sea"}) {
		t.Errorf("recent = %q, want Sea alone once no note has Wharf", cleared.Recent)
	}
}
