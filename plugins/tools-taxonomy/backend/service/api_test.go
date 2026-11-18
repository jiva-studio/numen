package service_test

import (
	"bufio"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/api"
)

const eventWait = 2 * time.Second

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

func readEvents(body io.Reader) <-chan api.TagMapMessage {
	events := make(chan api.TagMapMessage, 16)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(body)
		for scanner.Scan() {
			data, ok := strings.CutPrefix(scanner.Text(), "data: ")
			if !ok {
				continue
			}
			var message api.TagMapMessage
			if err := json.Unmarshal([]byte(data), &message); err != nil {
				return
			}
			events <- message
		}
	}()
	return events
}

func nextEvent(t *testing.T, events <-chan api.TagMapMessage) (api.TagMapMessage, bool) {
	t.Helper()
	select {
	case message, ok := <-events:
		if !ok {
			t.Error("the stream ended")
		}
		return message, ok
	case <-time.After(eventWait):
		t.Error("no message arrived")
		return api.TagMapMessage{}, false
	}
}

func TestGetTagMapAnswersTheTagsOfTheNotesThatHaveAny(t *testing.T) {
	p := newPlugin(t, map[string]string{
		"a.md": "---\ntags: [river, delta]\n---\n", "b.md": "---\ntags: lake\n---\n", "c.md": "# no tags\n",
	})
	web := p.serve(t)

	res := post(t, web, api.MethodGetTagMap)
	defer closeBody(t, res)

	var got api.TagMap
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil && res.StatusCode == http.StatusOK {
		t.Fatal(err)
	}
	want := map[string][]string{"a.md": {"river", "delta"}, "b.md": {"lake"}}
	if res.StatusCode != http.StatusOK || !maps.EqualFunc(got.Tags, want, slices.Equal) {
		t.Errorf("GetTagMap = %d %v, want 200 %v", res.StatusCode, got.Tags, want)
	}
}

func TestWatchTagMapSendsTheWholeMapThenTheChangedPaths(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	web := p.serve(t)
	res := post(t, web, api.MethodWatchTagMap)
	defer closeBody(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("WatchTagMap = %d, want 200", res.StatusCode)
	}
	events := readEvents(res.Body)

	first, ok := nextEvent(t, events)
	whole := map[string][]string{"a.md": {"river"}, "b.md": {"lake"}}
	if !ok || !maps.EqualFunc(first.Tags, whole, slices.Equal) || len(first.Changed) != 0 {
		t.Fatalf("first message = %+v, want the whole map %v", first, whole)
	}

	p.vault.File("a.md", "---\ntags: [river, delta]\n---\n")
	p.vault.Remove("b.md")
	p.deliver(t, "onFilesChanged:garden")

	second, ok := nextEvent(t, events)
	if !ok || len(second.Tags) != 0 || !slices.Equal(second.Changed["a.md"], []string{"river", "delta"}) {
		t.Fatalf("second message = %+v, want a.md changed", second)
	}
	if removed, has := second.Changed["b.md"]; !has || len(removed) != 0 || len(second.Changed) != 2 {
		t.Errorf("second message = %+v, want b.md with an empty list", second)
	}
}

func TestWatchTagMapOfAVaultWithoutTagsSendsAnEmptyMapFirst(t *testing.T) {
	p := newPlugin(t, map[string]string{"a.md": "# no tags\n"})
	web := p.serve(t)
	res := post(t, web, api.MethodWatchTagMap)
	defer closeBody(t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("WatchTagMap = %d, want 200", res.StatusCode)
	}

	first, ok := nextEvent(t, readEvents(res.Body))

	if !ok || len(first.Tags) != 0 || len(first.Changed) != 0 {
		t.Errorf("first message = %+v, want an empty map", first)
	}
}
