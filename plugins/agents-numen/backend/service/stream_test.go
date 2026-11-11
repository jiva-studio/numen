package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/chatapi"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/answer"
)

func TestStream_EmitsReplyChunksInOrder(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newTextReply("Water the garden today.")}}
	s := newServer(t, model.serve(t).URL)
	task := newTask("", "Plan?")
	task.Conversation = "talk-1"

	steps := ask(t, s, task)
	ask(t, s, task)

	if got, want := getSpoken(steps), []string{"Water ", "the ", "garden ", "today."}; !slices.Equal(got, want) {
		t.Errorf("said %q, want %q", got, want)
	}
	if got := describeSteps(steps); got != "model said stopped" || steps[len(steps)-1].GetStopped() != answer.ReasonDone {
		t.Errorf("steps are %q", got)
	}
	requests := model.getRequests()
	if !requests[0].Stream {
		t.Error("the request did not ask for a stream")
	}
	if msg := requests[1].Messages[2]; msg.Role != "assistant" || msg.Content == nil || *msg.Content != "Water the garden today." {
		t.Errorf("the conversation kept %+v", msg)
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
		replies: []conversation.Message{newTextReply("Both read.")},
	}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Read A and B."))

	if got := describeSteps(steps); got != "model said call finished call finished model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	if calls, want := vault.getCalls(), []string{`{"path":"A.md"}`, `{"path":"B.md"}`}; !slices.Equal(calls, want) {
		t.Errorf("the vault was called with %v, want %v", calls, want)
	}
	messages := model.getRequests()[1].Messages
	request := messages[len(messages)-3]
	if len(request.ToolCalls) != 2 || request.ToolCalls[0].ID != "c1" || request.ToolCalls[1].ID != "c2" ||
		request.ToolCalls[1].Function.Name != "note_read" || request.ToolCalls[1].Type != "function" ||
		request.Content == nil || *request.Content != "Reading both." {
		t.Errorf("the model received back %+v", request)
	}
	if messages[len(messages)-2].ToolCallID != "c1" || messages[len(messages)-1].ToolCallID != "c2" {
		t.Errorf("the results were told back as %+v", messages[len(messages)-2:])
	}
}

func TestAStreamedReplySplitOffAtItsLengthLimitCallsNoTool(t *testing.T) {
	model := &fakeChat{script: []string{
		`{"choices":[{"delta":{"content":"Reading "}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"note_read","arguments":"{\"pa"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"length"}]}`,
		`[DONE]`,
	}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Read."))

	if got := describeSteps(steps); got != "model said stopped" || steps[len(steps)-1].GetStopped() != answer.ReasonCut {
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
	s := newServer(t, model.serve(t).URL)

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
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != chatapi.ErrStreamEnded.Error() {
		t.Errorf("stopped %q", reason)
	}
}

func TestAStreamCompletedWithNoReasonToStopStopsTheRun(t *testing.T) {
	model := &fakeChat{script: []string{`{"choices":[{"delta":{"content":"The plan "}}]}`, `[DONE]`}}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != chatapi.ErrStreamEnded.Error() {
		t.Errorf("stopped %q", reason)
	}
}

func TestACanceledRunStopsReadingTheStream(t *testing.T) {
	released := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEvents(t, w, []string{`{"choices":[{"delta":{"content":"The plan "}}]}`})
		<-r.Context().Done()
		close(released)
	}))
	defer srv.Close()
	s := newServer(t, srv.URL)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var steps []*agentrun.Response
	err := s.Ask(ctx, newTask("", "Plan?"), func(r *agentrun.Response) error {
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
	hungUp := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEvents(t, w, []string{`{"choices":[{"delta":{"content":"Water the garden."}}]}`})
		<-r.Context().Done()
		close(hungUp)
	}))
	defer srv.Close()
	s := newServer(t, srv.URL)
	removed := errors.New("the window is gone")

	pieces := 0
	err := s.Ask(t.Context(), newTask("", "Plan?"), func(r *agentrun.Response) error {
		if r.GetText() == "" {
			return nil
		}
		pieces++
		return removed
	})

	if !errors.Is(err, removed) || pieces != 1 {
		t.Errorf("Ask returned %v after %d pieces", err, pieces)
	}
	<-hungUp
}

func TestStream_FallsBackToUnstreamedOnRejection(t *testing.T) {
	model := &fakeChat{
		rejectsStream: true,
		replies:       []conversation.Message{newCallReply("c", "note_read", `{"path":"A.md"}`), newTextReply("Water the garden.")},
	}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Plan?"))

	if got := describeSteps(steps); got != "model call finished model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	if got := getSpoken(steps); !slices.Equal(got, []string{"Water the garden."}) {
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
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if reason := steps[len(steps)-1].GetStopped(); !strings.Contains(reason, "400") || strings.Contains(reason, conversation.ErrStreamRejected.Error()) {
		t.Errorf("stopped %q", reason)
	}
}

func TestStream_EmitsWholeReplyWhenAPIIgnoresStreaming(t *testing.T) {
	model := &fakeChat{ignoresStream: true, replies: []conversation.Message{newTextReply("Water the garden.")}}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if got := getSpoken(steps); !slices.Equal(got, []string{"Water the garden."}) {
		t.Errorf("said %q", got)
	}
	if requests := model.getRequests(); len(requests) != 1 || !requests[0].Stream {
		t.Errorf("the requests were %+v", requests)
	}
}

func TestAStreamedToolCallAtAnImpossibleIndexStopsTheRun(t *testing.T) {
	model := &fakeChat{script: []string{
		fmt.Sprintf(`{"choices":[{"delta":{"tool_calls":[{"index":%d,"id":"c","function":{"name":"note_read"}}]}}]}`, chatapi.MaxStreamedCalls),
	}}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Read."))

	if reason := steps[len(steps)-1].GetStopped(); !strings.Contains(reason, "index") {
		t.Errorf("stopped %q", reason)
	}
}
