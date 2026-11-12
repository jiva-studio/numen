package service

import (
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func listWindowTools(t *testing.T) tools.Listing {
	t.Helper()
	handler, err := NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "window-token", "NUMEN_HOST_PORT": "1"})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	listing, err := tools.NewClient(hostcall.Endpoint{BaseURL: web.URL, Token: "window-token"}).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return listing
}

func getWindowToolSentences(listing tools.Listing) map[string][]string {
	byTool := map[string][]string{}
	for _, one := range listing.Tools {
		byTool[one.GetName()] = one.Instructions
	}
	return byTool
}

func TestTheWindowToolsHaveInstructionsOnShowingAPlace(t *testing.T) {
	byTool := getWindowToolSentences(listWindowTools(t))

	for _, name := range []string{"window_tab_list", "source_focus"} {
		if !slices.ContainsFunc(byTool[name], func(sentence string) bool { return strings.Contains(sentence, name) }) {
			t.Errorf("no instruction of %s names it: %q", name, byTool[name])
		}
	}
}

func TestEachWindowToolInstructionIsOneSentence(t *testing.T) {
	inside := regexp.MustCompile(`[.!?]\s+\S`)
	for name, sentences := range getWindowToolSentences(listWindowTools(t)) {
		for _, sentence := range sentences {
			if strings.TrimSpace(sentence) != sentence || !strings.HasSuffix(sentence, ".") || inside.MatchString(sentence) {
				t.Errorf("an instruction of %s is not one sentence: %q", name, sentence)
			}
		}
	}
}

func TestEveryWindowToolWithInstructionsNamesItself(t *testing.T) {
	for name, sentences := range getWindowToolSentences(listWindowTools(t)) {
		if len(sentences) == 0 {
			continue
		}
		named := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
		if !slices.ContainsFunc(sentences, named.MatchString) {
			t.Errorf("no instruction of %s names it: %q", name, sentences)
		}
	}
}

func TestTheWindowToolsThatShowAPlaceRequireWindowFocus(t *testing.T) {
	want := map[string][]string{
		"note_focus":          {"window:focus"},
		"source_focus":        {"window:focus"},
		"window_tab_activate": {"window:focus"},
		"window_tab_list":     {"window:focus"},
	}
	listing := listWindowTools(t)
	for name := range want {
		if !slices.ContainsFunc(listing.Tools, func(tool *tools.ToolInfo) bool { return tool.GetName() == name }) {
			t.Errorf("the window tools list no %s", name)
		}
	}

	for _, tool := range listing.Tools {
		if got := tool.Requires; !slices.Equal(got, want[tool.GetName()]) {
			t.Errorf("%s requires %q, want %q", tool.GetName(), got, want[tool.GetName()])
		}
	}
}

// The core no longer defines what "this note" means, so the tool that names the active tab does.
func TestTabListInstructionsDefineThisNoteAsTheActiveTabNote(t *testing.T) {
	sentences := getWindowToolSentences(listWindowTools(t))["window_tab_list"]

	if !slices.ContainsFunc(sentences, func(sentence string) bool {
		return strings.Contains(sentence, `"this note"`) && strings.Contains(sentence, "active tab") &&
			strings.Contains(sentence, "window_tab_list")
	}) {
		t.Errorf("no instruction of window_tab_list states that \"this note\" is the note of the active tab: %q", sentences)
	}
}
