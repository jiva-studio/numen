package agentbridge

import (
	"math"
	"reflect"
	"slices"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestConvertResponseStepCarriesTheCallNameAndEffect(t *testing.T) {
	effects := map[string]domain.StepKind{
		"read":   domain.StepRead,
		"edit":   domain.StepEdit,
		"remove": domain.StepRemove,
		"move":   domain.StepMove,
		"search": domain.StepSearch,
		"":       domain.StepToolCall,
		"other":  domain.StepToolCall,
	}
	for word, want := range effects {
		step := convertResponseStep(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_ToolCall{
			ToolCall: &v1.ToolCall{Tool: "Edit a note", Call: "toolu_01", Kind: word},
		}})
		if step.Kind != want || step.Call != "toolu_01" || step.Tool != "Edit a note" {
			t.Errorf("kind %q: got %+v", word, step)
		}
	}
}

// A limit on steps that no request can carry is the largest one it can, and a negative one is none.
func TestALimitOnStepsIsCachedWithinWhatTheRequestCarries(t *testing.T) {
	for limit, want := range map[int]int32{
		-5:            0,
		0:             0,
		7:             7,
		math.MaxInt32: math.MaxInt32,
		math.MaxInt:   math.MaxInt32,
	} {
		if got := newAskRequest(agent.Task{MaxSteps: limit}).GetMaxSteps(); got != want {
			t.Errorf("%d was sent as %d, want %d", limit, got, want)
		}
	}
}

// Everything the task provides to the agent is present in the request, and the
// effect of each tool is encoded in the word the plugins read.
func TestTheRequestCarriesWhatTheTaskProducesTheAgent(t *testing.T) {
	got := newAskRequest(agent.Task{
		Question: "q", Focus: "f", Conversation: "c", Agent: "one", Model: "m", Reasoning: "high", Instructions: "i",
		Tools: []string{"note_edit"}, MaxSteps: 7, Root: "/vault", Command: []string{"/bin/one"},
		Endpoint: agent.ToolEndpoint{URL: "http://127.0.0.1:1/mcp", Token: "t"},
		Context:  map[string]string{"org.example.sample/item": "k7"},
		ToolDeclarations: []agent.ToolDeclaration{
			{Name: "note_edit", Title: "Edit a note", Kind: domain.StepEdit, About: "path", Match: "stood", Text: "becomes"},
			{Name: "note_read", Kind: domain.StepToolCall, Element: "path"},
		},
	})

	if got.GetPrompt() != "q" || got.GetFocus() != "f" || got.GetConversation() != "c" || got.GetAgent() != "one" ||
		got.GetModel() != "m" || got.GetReasoning() != "high" || got.GetInstructions() != "i" || got.GetMaxSteps() != 7 ||
		!slices.Equal(got.GetTools(), []string{"note_edit"}) {
		t.Errorf("the question is %v", got)
	}
	if got.GetRoot() != "/vault" || !slices.Equal(got.GetCommand(), []string{"/bin/one"}) ||
		got.GetToolsUrl() != "http://127.0.0.1:1/mcp" || got.GetToolsToken() != "t" ||
		got.GetContext()["org.example.sample/item"] != "k7" {
		t.Errorf("what the agent is given is %v", got)
	}
	declared := got.GetToolDeclarations()
	if len(declared) != 2 {
		t.Fatalf("the declarations are %v", declared)
	}
	if edit := declared[0]; edit.GetName() != "note_edit" || edit.GetTitle() != "Edit a note" ||
		edit.GetKind() != "edit" || edit.GetAbout() != "path" || edit.GetMatch() != "stood" || edit.GetText() != "becomes" {
		t.Errorf("the edit tool is declared as %v", edit)
	}
	if read := declared[1]; read.GetKind() != "" || read.GetElement() != "path" {
		t.Errorf("the other tool is declared as %v", read)
	}
}

// The word that encodes a tool's effect is the word it is decoded from.
func TestAnEffectIsReportedAndReadInTheSameWord(t *testing.T) {
	for _, effect := range []domain.StepKind{
		domain.StepRead, domain.StepEdit, domain.StepRemove, domain.StepMove, domain.StepSearch, domain.StepToolCall,
	} {
		if got := getStepKind(getKindWord(effect)); got != effect {
			t.Errorf("%v received as %q and read as %v", effect, getKindWord(effect), got)
		}
	}
}

// A change in progress arrives at the core as a draft of the note, with the
// text it replaces and the replacement text so far.
func TestAChangeBeingSavedIsADraft(t *testing.T) {
	got := convertResponseStep(&v1.AskAgentResponse{Step: &v1.AskAgentResponse_EditProgress{
		EditProgress: &v1.EditProgress{Call: "toolu_01", Path: "a/b.md", Match: "old", Text: "new"},
	}})

	want := agent.Step{
		Kind: domain.StepDraft, Call: "toolu_01", Place: domain.Place{Path: "a/b.md"}, Match: "old", Text: "new",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
