package service_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/api"
)

func (p plugin) serve(t *testing.T) *httptest.Server {
	t.Helper()
	handler, err := servertest.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		p.app.Mount(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return web
}

func post(t *testing.T, web *httptest.Server, method string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, web.URL+"/"+api.Service+"/"+method, strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer plugin-token")
	req.Header.Set("Numen-Vault", "garden")
	res, err := web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func closeBody(t *testing.T, res *http.Response) {
	t.Helper()
	if err := res.Body.Close(); err != nil {
		t.Log(err)
	}
}

func readTagMap(t *testing.T, web *httptest.Server) api.TagMap {
	t.Helper()
	res := post(t, web, api.MethodGetTagMap)
	defer closeBody(t, res)
	var got api.TagMap
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil && res.StatusCode == http.StatusOK {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GetTagMap = %d, want 200", res.StatusCode)
	}
	return got
}

func TestGetTagMapAnswersTheTagsOfTheNotesThatHaveAny(t *testing.T) {
	p := newPlugin(t, map[string]string{
		"a.md": "---\ntags: [river, delta]\n---\n", "b.md": "---\ntags: lake\n---\n", "c.md": "# no tags\n",
	})
	web := p.serve(t)

	got := readTagMap(t, web)

	want := map[string][]string{"a.md": {"river", "delta"}, "b.md": {"lake"}}
	if !maps.EqualFunc(got.Tags, want, slices.Equal) {
		t.Errorf("GetTagMap = %v, want %v", got.Tags, want)
	}
}

func TestGetTagMapAnswersTheUpdatedPathsAfterFilesChange(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	web := p.serve(t)
	readTagMap(t, web)

	p.vault.WriteFile("a.md", "---\ntags: [river, delta]\n---\n")
	p.vault.Remove("b.md")
	p.deliver(t, "onFilesChanged:garden")

	got := readTagMap(t, web)
	want := map[string][]string{"a.md": {"river", "delta"}}
	if !maps.EqualFunc(got.Tags, want, slices.Equal) {
		t.Errorf("GetTagMap = %v, want %v", got.Tags, want)
	}
}

func TestGetTagMapOfAVaultWithoutTagsAnswersAnEmptyMap(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "# no tags\n"})
	web := p.serve(t)

	got := readTagMap(t, web)

	if got.Tags == nil || len(got.Tags) != 0 {
		t.Errorf("GetTagMap = %v, want an empty map", got.Tags)
	}
}
