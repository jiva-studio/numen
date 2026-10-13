package agent

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var testAt = []string{"zeta", "model"}

// names is the name of each row, in the order drawn.
func names(models []port.Model) []string {
	var got []string
	for _, one := range models {
		got = append(got, one.Name)
	}
	return got
}

// defaults is the name of each row that is the default.
func defaults(models []port.Model) []string {
	var got []string
	for _, one := range models {
		if one.IsDefault {
			got = append(got, one.Name)
		}
	}
	return got
}

func TestAnAgentThatReportedNothingOffersTheModelTheSettingsNameAlone(t *testing.T) {
	got := getAgentModels(testAt, "house-model", nil)
	if !slices.Equal(names(got), []string{"house-model"}) || !slices.Equal(defaults(got), []string{"house-model"}) {
		t.Fatalf("offered %+v", got)
	}
	if got[0].Presence != port.NothingToFetch || got[0].Writes[0].JSON != `"house-model"` {
		t.Errorf("the row is %+v", got[0])
	}
}

func TestAnAgentThatReportedNothingAndHasNoSettingOffersItsOwnDefault(t *testing.T) {
	got := getAgentModels(testAt, "", nil)
	if len(got) != 1 || got[0].Name != "" || !got[0].IsDefault || got[0].Writes[0].JSON != `""` {
		t.Fatalf("offered %+v", got)
	}
}

func TestTheModelsAnAgentReportedAreOfferedInItsOrder(t *testing.T) {
	offered := []AgentModel{{Name: "zeta-1", Title: "Zeta One"}, {Name: "zeta-2"}, {Name: "zeta-3", IsDefault: true}}

	got := getAgentModels(testAt, "", offered)
	if !slices.Equal(names(got), []string{"zeta-1", "zeta-2", "zeta-3"}) || got[0].Title != "Zeta One" || got[1].Title != "zeta-2" {
		t.Fatalf("offered %+v", got)
	}
	if !slices.Equal(defaults(got), []string{"zeta-3"}) {
		t.Errorf("the agent names its default, defaults are %q", defaults(got))
	}

	got = getAgentModels(testAt, "", offered[:2])
	if !slices.Equal(defaults(got), []string{"zeta-1"}) {
		t.Errorf("the first is the default where none is named, defaults are %q", defaults(got))
	}

	got = getAgentModels(testAt, "zeta-2", offered)
	if !slices.Equal(defaults(got), []string{"zeta-2"}) {
		t.Errorf("the model the settings name is the default, defaults are %q", defaults(got))
	}
}

func TestAModelTheSettingsNameThatTheAgentDidNotReportIsListedAfterThem(t *testing.T) {
	got := getAgentModels(testAt, "house-model", []AgentModel{{Name: "zeta-1"}})
	if !slices.Equal(names(got), []string{"zeta-1", "house-model"}) {
		t.Fatalf("offered %q", names(got))
	}
	if got[1].Group != groupConfigured || got[0].Group != groupInFull {
		t.Errorf("groups are %q and %q", got[0].Group, got[1].Group)
	}
	if !slices.Equal(defaults(got), []string{"zeta-1"}) {
		t.Errorf("defaults are %q", defaults(got))
	}
}

// fakeSource answers with what it holds and counts the questions.
type fakeSource struct {
	models []AgentModel
	err    error
	asked  int
}

func (s *fakeSource) GetModels(context.Context) ([]AgentModel, error) {
	s.asked++
	return s.models, s.err
}

func TestAnAgentIsAskedOncePerSessionAndAgainWhenToldTo(t *testing.T) {
	kept := &catalogue{}
	source := &fakeSource{models: []AgentModel{{Name: "zeta-1"}}}

	for range 3 {
		if got := kept.get("zeta", "k", source); len(got) != 1 {
			t.Fatalf("got %+v", got)
		}
	}
	if source.asked != 1 {
		t.Errorf("asked %d times", source.asked)
	}
	kept.get("zeta", "other", source)
	if source.asked != 2 {
		t.Errorf("asked %d times after the way of asking changed", source.asked)
	}
	kept.clear()
	kept.get("zeta", "other", source)
	if source.asked != 3 {
		t.Errorf("asked %d times after a refresh", source.asked)
	}
}

func TestAnAgentThatFailsToAnswerOffersNoModelsAndIsNotAskedAgain(t *testing.T) {
	kept := &catalogue{}
	source := &fakeSource{models: []AgentModel{{Name: "zeta-1"}}, err: errors.New("no answer")}

	if got := kept.get("zeta", "k", source); got != nil {
		t.Errorf("got %+v", got)
	}
	kept.get("zeta", "k", source)
	if source.asked != 1 {
		t.Errorf("asked %d times", source.asked)
	}
}

func TestEveryAgentListedIsAskedForItsOwnModels(t *testing.T) {
	got := getModels(&catalogue{}, []listed{
		{use: "zeta", at: []string{"zeta", "model"}, source: &fakeSource{models: []AgentModel{{Name: "zeta-1"}, {Name: "zeta-2"}}}},
		{use: "omega", at: []string{"omega", "model"}, set: "omega-set", source: &fakeSource{}},
		{use: "sigma", at: []string{"sigma", "model"}, source: &fakeSource{err: errors.New("down")}},
	})
	if want := []string{"zeta-1", "zeta-2", "omega-set", ""}; !slices.Equal(names(got), want) {
		t.Fatalf("offered %q, want %q", names(got), want)
	}
}

func TestClaudeModelsAreListedFromTheModelsEndpointWithTheKey(t *testing.T) {
	var key, version string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, version = r.Header.Get("x-api-key"), r.Header.Get("anthropic-version")
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"zeta-big","display_name":"Zeta Big"},{"id":"","display_name":"Nameless"},{"id":"zeta-small"}]}`)
	}))
	defer server.Close()

	got, err := claudeModels{key: "k-1", base: server.URL + "/"}.GetModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != (AgentModel{Name: "zeta-big", Title: "Zeta Big"}) || got[1].Name != "zeta-small" {
		t.Errorf("listed %+v", got)
	}
	if key != "k-1" || version == "" {
		t.Errorf("asked with key %q and version %q", key, version)
	}
}

func TestClaudeModelsWithoutAKeyAskNothing(t *testing.T) {
	asked := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { asked = true }))
	defer server.Close()

	got, err := claudeModels{base: server.URL}.GetModels(context.Background())
	if got != nil || err != nil || asked {
		t.Errorf("got %+v, %v, asked %v", got, err, asked)
	}
}

func TestClaudeModelsRefusedByTheEndpointAreAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	if _, err := (claudeModels{key: "bad", base: server.URL}).GetModels(context.Background()); err == nil {
		t.Error("a refusal was taken for an answer")
	}
}

// serveCodex plays an app server over pipes, answering each request with the result of reply.
func serveCodex(t *testing.T, reply func(method string) string) (io.Writer, io.Reader) {
	t.Helper()
	inRead, inWrite := io.Pipe()
	outRead, outWrite := io.Pipe()
	go func() {
		defer outWrite.Close()
		lines := bufio.NewScanner(inRead)
		for lines.Scan() {
			text := lines.Text()
			method := text[strings.Index(text, `"method":"`)+len(`"method":"`):]
			method = method[:strings.Index(method, `"`)]
			if !strings.Contains(text, `"id"`) {
				continue
			}
			id := "1"
			if method == "model/list" {
				id = "2"
			}
			_, _ = io.WriteString(outWrite, `{"method":"noise","params":{}}`+"\n")
			_, _ = io.WriteString(outWrite, `{"id":`+id+`,`+reply(method)+"}\n")
		}
	}()
	t.Cleanup(func() { inWrite.Close() })
	return inWrite, outRead
}

func TestCodexModelsAreListedByTheAppServerWithoutTheHiddenOnes(t *testing.T) {
	in, out := serveCodex(t, func(method string) string {
		if method == "initialize" {
			return `"result":{}`
		}
		return `"result":{"data":[` +
			`{"id":"a","model":"omega-1","displayName":"Omega One","hidden":false,"isDefault":true},` +
			`{"id":"b","model":"omega-hidden","hidden":true},` +
			`{"id":"omega-2","model":"","displayName":"Omega Two"}],"nextCursor":null}`
	})

	got, err := listCodexModels(in, out)
	if err != nil {
		t.Fatal(err)
	}
	want := []AgentModel{{Name: "omega-1", Title: "Omega One", IsDefault: true}, {Name: "omega-2", Title: "Omega Two"}}
	if !slices.Equal(got, want) {
		t.Errorf("listed %+v, want %+v", got, want)
	}
}

func TestCodexModelsTheAppServerRefusesAreAnError(t *testing.T) {
	in, out := serveCodex(t, func(string) string { return `"error":{"code":-1,"message":"not now"}` })

	if _, err := listCodexModels(in, out); err == nil || !strings.Contains(err.Error(), "not now") {
		t.Errorf("error is %v", err)
	}
}

func TestCodexModelsAnAppServerThatEndsIsAnError(t *testing.T) {
	if _, err := listCodexModels(io.Discard, strings.NewReader("")); err == nil {
		t.Error("an app server that said nothing was taken for an answer")
	}
}

func TestCodexModelsAreReadFromItsCacheWithoutTheHiddenOnes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	cache := `{"models":[
		{"slug":"alpha","display_name":"Alpha One","visibility":"list"},
		{"slug":"secret","display_name":"Secret","visibility":"hide"},
		{"slug":"","display_name":"Nameless","visibility":"list"},
		{"slug":"beta","display_name":"","visibility":"list"}
	]}`
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(cache), 0o644); err != nil {
		t.Fatal(err)
	}

	want := []AgentModel{{Name: "alpha", Title: "Alpha One"}, {Name: "beta"}}
	if got := readCodexCache(); !slices.Equal(got, want) {
		t.Errorf("listed %+v, want %+v", got, want)
	}

	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(`{"models":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readCodexCache(); got != nil {
		t.Errorf("a cache of the wrong shape listed %+v", got)
	}
}

func TestAntigravityReportsNoModels(t *testing.T) {
	if got, err := (antigravityModels{}).GetModels(context.Background()); got != nil || err != nil {
		t.Errorf("got %+v, %v", got, err)
	}
}
