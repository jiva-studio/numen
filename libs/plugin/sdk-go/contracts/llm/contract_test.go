package llm_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// inProcess is a connection.Caller that runs the operations of a provider in the test, for one connection.
type inProcess struct {
	provider   connection.Provider
	call       connection.Call
	operations []string
	priorities []connection.Priority
}

func (c *inProcess) Call(ctx context.Context, operation string, payload []byte, priority connection.Priority) (connection.Frames, error) {
	c.operations, c.priorities = append(c.operations, operation), append(c.priorities, priority)
	serve, isServed := c.provider[operation]
	if !isServed {
		return &frames{err: connection.ErrNoConnection}, nil
	}
	got := &frames{}
	got.done, got.err = serve(ctx, c.call, payload, func(data []byte) error {
		got.data = append(got.data, data)
		return nil
	})
	return got, nil
}

type frames struct {
	data [][]byte
	next int
	done []byte
	err  error
}

func (f *frames) Next() bool {
	if f.next >= len(f.data) {
		return false
	}
	f.next++
	return true
}

func (f *frames) Data() []byte { return f.data[f.next-1] }

func (f *frames) Done() []byte { return f.done }

func (f *frames) Err() error { return f.err }

func (f *frames) Close() error { return nil }

// assistant is a provider of every role of the contract that answers from what it received.
type assistant struct {
	request *llm.ChatRequest
}

func (a *assistant) Complete(_ context.Context, _ connection.Call, request *llm.CompleteRequest) (*llm.CompleteResult, error) {
	return &llm.CompleteResult{Text: "fixed: " + request.GetMessages()[len(request.GetMessages())-1].GetText()}, nil
}

func (a *assistant) Chat(_ context.Context, _ connection.Call, request *llm.ChatRequest, send func(*llm.ChatFrame) error) (*llm.ChatFinish, error) {
	a.request = request
	if err := send(&llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: "Let me look."}}); err != nil {
		return nil, err
	}
	call := &llm.ToolCall{Id: "c1", Name: "list_notes", ArgumentsJson: `{"folder":"Trips"}`}
	if err := send(&llm.ChatFrame{Event: &llm.ChatFrame_ToolCall{ToolCall: call}}); err != nil {
		return nil, err
	}
	return &llm.ChatFinish{Reason: llm.FinishReason_FINISH_REASON_TOOL_CALLS}, nil
}

func (a *assistant) Embed(_ context.Context, _ connection.Call, request *llm.EmbedRequest) (*llm.EmbedResult, error) {
	result := &llm.EmbedResult{}
	for _, text := range request.GetTexts() {
		result.Vectors = append(result.Vectors, &llm.Vector{Values: []float32{float32(len(text)), 1}})
	}
	return result, nil
}

func (a *assistant) GetProfile(_ context.Context, call connection.Call) (*llm.Profile, error) {
	return &llm.Profile{Model: call.Options[llm.OptionModel], Source: "server https://models.test/v1", Width: 768, MaxTokens: 512}, nil
}

func (a *assistant) ListModels(_ context.Context, _ connection.Call, query connection.OptionsQuery) (connection.ListedOptions, error) {
	return connection.ListedOptions{Options: []connection.Option{{Value: "small", Title: "Small " + query.Operations[0]}}}, nil
}

func (a *assistant) ListReasoningLevels(_ context.Context, call connection.Call) ([]connection.Option, error) {
	return []connection.Option{{Value: "low", Title: "Low for " + call.Options[llm.OptionModel]}}, nil
}

func newCaller(roles llm.Roles) *inProcess {
	return &inProcess{
		provider: llm.Provide(nil, roles),
		call:     connection.Call{Type: "server", Options: map[string]string{llm.OptionModel: "nomic-embed-text"}},
	}
}

func TestCompleteReachesTheCompleterAndReturnsItsText(t *testing.T) {
	a := &assistant{}
	caller := newCaller(llm.Roles{Completer: a})

	result, err := llm.Complete(t.Context(), caller, &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: "a sleepy cat"}}}, connection.PriorityInteractive)

	if err != nil || result.GetText() != "fixed: a sleepy cat" {
		t.Fatalf("Complete = %v, %v", result, err)
	}
	if !slices.Equal(caller.operations, []string{llm.OperationComplete}) || caller.priorities[0] != connection.PriorityInteractive {
		t.Errorf("operations = %v, priorities = %v", caller.operations, caller.priorities)
	}
}

func TestChatStreamsTheFramesOfTheTurnAndEndsWithTheFinish(t *testing.T) {
	a := &assistant{}
	caller := newCaller(llm.Roles{Chatter: a})

	stream, err := llm.Chat(t.Context(), caller, &llm.ChatRequest{
		Messages: []*llm.ChatMessage{{Role: "user", Text: "what is in Trips?"}},
		Tools:    []*llm.ToolDefinition{{Name: "list_notes", ParametersJson: `{"type":"object"}`}},
	}, connection.PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	var text string
	var calls []string
	for stream.Next() {
		text += stream.Frame().GetTextDelta()
		if call := stream.Frame().GetToolCall(); call != nil {
			calls = append(calls, call.GetName())
		}
	}

	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if text != "Let me look." || !slices.Equal(calls, []string{"list_notes"}) || stream.Finish().GetReason() != llm.FinishReason_FINISH_REASON_TOOL_CALLS {
		t.Errorf("text = %q, calls = %v, finish = %v", text, calls, stream.Finish())
	}
	if a.request.GetTools()[0].GetName() != "list_notes" || a.request.GetMessages()[0].GetText() != "what is in Trips?" {
		t.Errorf("request = %v", a.request)
	}
}

func TestEmbedReturnsOneVectorForEachText(t *testing.T) {
	caller := newCaller(llm.Roles{Embedder: &assistant{}})

	result, err := llm.Embed(t.Context(), caller, &llm.EmbedRequest{Texts: []string{"ab", "abcd"}}, connection.PriorityBackground)

	if err != nil || len(result.GetVectors()) != 2 || result.GetVectors()[1].GetValues()[0] != 4 {
		t.Errorf("Embed = %v, %v", result, err)
	}
}

// oneVector embeds every request into one vector, whatever the number of its texts.
type oneVector struct{}

func (oneVector) Embed(context.Context, connection.Call, *llm.EmbedRequest) (*llm.EmbedResult, error) {
	return &llm.EmbedResult{Vectors: []*llm.Vector{{Values: []float32{1}}}}, nil
}

func TestEmbedOfAnotherNumberOfVectorsThanTextsIsAnError(t *testing.T) {
	caller := newCaller(llm.Roles{Embedder: oneVector{}})

	result, err := llm.Embed(t.Context(), caller, &llm.EmbedRequest{Texts: []string{"quay", "jetty"}}, connection.PriorityBackground)

	if err == nil || result != nil {
		t.Errorf("Embed = %v, %v, want an error", result, err)
	}
}

func TestEmbedOfNoTextCallsNoProvider(t *testing.T) {
	caller := newCaller(llm.Roles{Embedder: &assistant{}})

	result, err := llm.Embed(t.Context(), caller, &llm.EmbedRequest{}, connection.PriorityBackground)

	if err != nil || len(result.GetVectors()) != 0 || len(caller.operations) != 0 {
		t.Errorf("Embed = %v, %v, operations = %v", result, err, caller.operations)
	}
}

func TestGetProfileIsBackgroundAndNamesTheModelOfTheRow(t *testing.T) {
	caller := newCaller(llm.Roles{Profiler: &assistant{}})

	profile, err := llm.GetProfile(t.Context(), caller)

	if err != nil || profile.GetModel() != "nomic-embed-text" || profile.GetWidth() != 768 || profile.GetMaxTokens() != 512 {
		t.Fatalf("GetProfile = %v, %v", profile, err)
	}
	if caller.priorities[0] != connection.PriorityBackground {
		t.Errorf("priority = %v, want background", caller.priorities[0])
	}
}

func TestOnlyTheRolesGivenAreServed(t *testing.T) {
	provider := llm.Provide(connection.Provider{"acp.task": nil}, llm.Roles{Models: &assistant{}, Reasoning: &assistant{}})

	var names []string
	for name := range provider {
		names = append(names, name)
	}
	slices.Sort(names)

	if !slices.Equal(names, []string{"acp.task", llm.OperationModels, llm.OperationReasoning}) {
		t.Errorf("operations = %v", names)
	}
}

func TestAFailureOfTheProviderIsTheErrorOfTheCall(t *testing.T) {
	caller := newCaller(llm.Roles{})

	_, err := llm.Complete(t.Context(), caller, &llm.CompleteRequest{}, connection.PriorityBackground)

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want the error of the call", err)
	}
}

func TestAPayloadThatIsNoRequestOfTheContractIsAnInvalidRequest(t *testing.T) {
	serve := llm.Provide(nil, llm.Roles{Completer: &assistant{}})[llm.OperationComplete]

	_, err := serve(t.Context(), connection.Call{}, []byte{0xff, 0xff}, func([]byte) error { return nil })

	if !connection.IsStop(err) || err.Error() != connection.ReasonInvalidRequest.String() {
		t.Errorf("err = %v, want a stop of an invalid request", err)
	}
}
