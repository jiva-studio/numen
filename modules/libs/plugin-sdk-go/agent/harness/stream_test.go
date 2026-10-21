package harness

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestTheReplyIsSaidPieceByPieceInTheOrderItIsWritten(t *testing.T) {
	model := &fakeChat{replies: []message{newTextReply("Water the garden today.")}}
	s := newServer(model.serve(t).URL)
	task := newTask("", "Plan?")
	task.Conversation = "talk-1"

	steps := ask(t, s, task)
	ask(t, s, task)

	if got, want := getSaid(steps), []string{"Water ", "the ", "garden ", "today."}; !slices.Equal(got, want) {
		t.Errorf("said %q, want %q", got, want)
	}
	if got := describeSteps(steps); got != "model said stopped" || steps[len(steps)-1].GetStopped() != reasonDone {
		t.Errorf("steps are %q", got)
	}
	requests := model.getRequests()
	if !requests[0].Stream {
		t.Error("the request did not ask for a stream")
	}
	if kept := requests[1].Messages[2]; kept.Role != "assistant" || kept.Content == nil || *kept.Content != "Water the garden today." {
		t.Errorf("the conversation kept %+v", kept)
	}
}

func TestStreamedToolCallFragmentsAreAssembledByIndex(t *testing.T) {
	model := &fakeChat{
		script: []string{
			`{"choices":[{"delta":{"role":"assistant","content":"Reading both."}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"note_read","arguments":""}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"c2","type":"function","function":{"name":"note_read","arguments":"{\"pa"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"A.md\"}"}}]}}]}`,
			`{"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"th\":\"B.md\"}"}}]}}]}`,
			`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			`[DONE]`,
		},
		replies: []message{newTextReply("Both read.")},
	}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Read A and B."))

	if got := describeSteps(steps); got != "model said call finished call finished model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	if calls, want := vault.getCalls(), []string{`{"path":"A.md"}`, `{"path":"B.md"}`}; !slices.Equal(calls, want) {
		t.Errorf("the vault was called with %v, want %v", calls, want)
	}
	told := model.getRequests()[1].Messages
	asked := told[len(told)-3]
	if len(asked.ToolCalls) != 2 || asked.ToolCalls[0].ID != "c1" || asked.ToolCalls[1].ID != "c2" ||
		asked.ToolCalls[1].Function.Name != "note_read" || asked.ToolCalls[1].Type != "function" ||
		asked.Content == nil || *asked.Content != "Reading both." {
		t.Errorf("the model was told back %+v", asked)
	}
	if told[len(told)-2].ToolCallID != "c1" || told[len(told)-1].ToolCallID != "c2" {
		t.Errorf("the results were told back as %+v", told[len(told)-2:])
	}
}

func TestAStreamedReplyCutOffAtItsLengthLimitCallsNoTool(t *testing.T) {
	model := &fakeChat{script: []string{
		`{"choices":[{"delta":{"content":"Reading "}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"note_read","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"length"}]}`,
		`[DONE]`,
	}}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Read."))

	if got := describeSteps(steps); got != "model said stopped" || steps[len(steps)-1].GetStopped() != reasonCut {
		t.Errorf("steps are %q, stopped %q", got, steps[len(steps)-1].GetStopped())
	}
	if calls := vault.getCalls(); len(calls) != 0 {
		t.Errorf("the vault was called with %v", calls)
	}
}

func TestAnErrorInTheStreamStopsTheRunWithItsReason(t *testing.T) {
	model := &fakeChat{script: []string{
		`{"choices":[{"delta":{"content":"The plan "}}]}`,
		`{"error":{"message":"the provider is overloaded","code":502}}`,
	}}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if got := describeSteps(steps); got != "model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	if reason := steps[len(steps)-1].GetStopped(); !strings.Contains(reason, "the provider is overloaded") {
		t.Errorf("stopped %q", reason)
	}
}

func TestAStreamThatEndsBeforeTheReplyStopsTheRun(t *testing.T) {
	model := &fakeChat{script: []string{`{"choices":[{"delta":{"content":"The plan "}}]}`}}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != errStreamEnded.Error() {
		t.Errorf("stopped %q", reason)
	}
}

func TestAStreamDoneWithNoReasonToStopStopsTheRun(t *testing.T) {
	model := &fakeChat{script: []string{`{"choices":[{"delta":{"content":"The plan "}}]}`, `[DONE]`}}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != errStreamEnded.Error() {
		t.Errorf("stopped %q", reason)
	}
}

func TestACancelledRunStopsReadingTheStream(t *testing.T) {
	released := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEvents(w, []string{`{"choices":[{"delta":{"content":"The plan "}}]}`})
		<-r.Context().Done()
		close(released)
	}))
	defer srv.Close()
	s := newServer(srv.URL)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var steps []*numenv1.AskAgentResponse
	err := s.Ask(ctx, newTask("", "Plan?"), func(r *numenv1.AskAgentResponse) error {
		steps = append(steps, r)
		if r.GetText() != "" {
			cancel()
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if got := describeSteps(steps); got != "model said" {
		t.Errorf("steps are %q", got)
	}
	<-released
}

func TestAStepThatCannotBeSentEndsTheStream(t *testing.T) {
	model := &fakeChat{replies: []message{newTextReply("Water the garden.")}}
	s := newServer(model.serve(t).URL)
	gone := errors.New("the window is gone")

	pieces := 0
	err := s.Ask(t.Context(), newTask("", "Plan?"), func(r *numenv1.AskAgentResponse) error {
		if r.GetText() == "" {
			return nil
		}
		pieces++
		return gone
	})

	if !errors.Is(err, gone) || pieces != 1 {
		t.Errorf("Ask returned %v after %d pieces", err, pieces)
	}
}

func TestAChatAPIThatRefusesToStreamIsAskedUnstreamed(t *testing.T) {
	model := &fakeChat{
		refusesStream: true,
		replies:       []message{newCallReply("c", "note_read", `{"path":"A.md"}`), newTextReply("Water the garden.")},
	}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Plan?"))

	if got := describeSteps(steps); got != "model call finished model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	if got := getSaid(steps); !slices.Equal(got, []string{"Water the garden."}) {
		t.Errorf("said %q", got)
	}
	var streamed []bool
	for _, one := range model.getRequests() {
		streamed = append(streamed, one.Stream)
	}
	if want := []bool{true, false, false}; !slices.Equal(streamed, want) {
		t.Errorf("the requests streamed %v, want %v", streamed, want)
	}
}

func TestABadRequestIsReportedWhenTheUnstreamedOneFailsToo(t *testing.T) {
	model := &fakeChat{status: http.StatusBadRequest}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if reason := steps[len(steps)-1].GetStopped(); !strings.Contains(reason, "400") || strings.Contains(reason, errStreamRefused.Error()) {
		t.Errorf("stopped %q", reason)
	}
}

func TestAChatAPIThatIgnoresTheStreamHasItsWholeReplySaid(t *testing.T) {
	model := &fakeChat{ignoresStream: true, replies: []message{newTextReply("Water the garden.")}}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if got := getSaid(steps); !slices.Equal(got, []string{"Water the garden."}) {
		t.Errorf("said %q", got)
	}
	if requests := model.getRequests(); len(requests) != 1 || !requests[0].Stream {
		t.Errorf("the requests were %+v", requests)
	}
}

func TestAStreamedToolCallAtAnImpossibleIndexStopsTheRun(t *testing.T) {
	model := &fakeChat{script: []string{
		fmt.Sprintf(`{"choices":[{"delta":{"tool_calls":[{"index":%d,"id":"c","function":{"name":"note_read"}}]}}]}`, maxStreamedCalls),
	}}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Read."))

	if reason := steps[len(steps)-1].GetStopped(); !strings.Contains(reason, "index") {
		t.Errorf("stopped %q", reason)
	}
}
