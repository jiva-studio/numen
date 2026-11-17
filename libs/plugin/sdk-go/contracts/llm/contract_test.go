package llm_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	numenv1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
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

// Rerank scores each document by the number of its bytes.
func (a *assistant) Rerank(_ context.Context, _ connection.Call, request *numenv1.RerankRequest) (*numenv1.RerankResult, error) {
	result := &numenv1.RerankResult{}
	for _, document := range request.GetDocuments() {
		result.Scores = append(result.Scores, float32(len(document)))
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
	firstFinish := stream.Finish()
	if text != "Let me look." || !slices.Equal(calls, []string{"list_notes"}) || firstFinish.GetReason() != llm.FinishReason_FINISH_REASON_TOOL_CALLS {
		t.Errorf("text = %q, calls = %v, finish = %v", text, calls, firstFinish)
	}
	if stream.Finish() != firstFinish {
		t.Error("Finish did not return the cached finish on second call")
	}
	if err := stream.Close(); err != nil {
		t.Errorf("Close = %v", err)
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

	if err != nil || result == nil || len(result.GetVectors()) != 0 || len(caller.operations) != 0 {
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

	if names := getOperations(provider); !slices.Equal(names, []string{"acp.task", llm.OperationModels, llm.OperationReasoning}) {
		t.Errorf("operations = %v", names)
	}
}

func TestRerankIsServedOnlyWithAReranker(t *testing.T) {
	without := llm.Provide(nil, llm.Roles{Embedder: &assistant{}})
	with := llm.Provide(nil, llm.Roles{Embedder: &assistant{}, Reranker: &assistant{}})

	if names := getOperations(without); !slices.Equal(names, []string{llm.OperationEmbed}) {
		t.Errorf("operations without a reranker = %v", names)
	}
	if names := getOperations(with); !slices.Equal(names, []string{llm.OperationEmbed, "llm.rerank"}) || llm.OperationRerank != "llm.rerank" {
		t.Errorf("operations with a reranker = %v", names)
	}
}

func getOperations(provider connection.Provider) []string {
	var names []string
	for name := range provider {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

type staticCaller struct {
	frames *frames
	err    error
}

func (c *staticCaller) Call(context.Context, string, []byte, connection.Priority) (connection.Frames, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.frames, nil
}

type failingAssistant struct {
	err   error
	frame *llm.ChatFrame
}

func (f *failingAssistant) Complete(context.Context, connection.Call, *llm.CompleteRequest) (*llm.CompleteResult, error) {
	return nil, f.err
}

func (f *failingAssistant) Chat(_ context.Context, _ connection.Call, _ *llm.ChatRequest, send func(*llm.ChatFrame) error) (*llm.ChatFinish, error) {
	return nil, send(f.frame)
}

func TestAFailureOfTheProviderIsTheErrorOfTheCall(t *testing.T) {
	encodeCaller := newCaller(llm.Roles{Completer: &assistant{}, Embedder: &assistant{}, Chatter: &assistant{}})
	if _, err := llm.Complete(
		t.Context(), encodeCaller, &llm.CompleteRequest{Messages: []*llm.Message{{Text: "\xff"}}}, connection.PriorityBackground,
	); err == nil || !strings.Contains(err.Error(), "llm: encode "+llm.OperationComplete) {
		t.Errorf("Complete invalid UTF-8 err = %v", err)
	}
	if _, err := llm.Embed(
		t.Context(), encodeCaller, &llm.EmbedRequest{Texts: []string{"\xff"}}, connection.PriorityBackground,
	); err == nil || !strings.Contains(err.Error(), "llm: encode "+llm.OperationEmbed) {
		t.Errorf("Embed invalid UTF-8 err = %v", err)
	}
	if _, err := llm.Chat(
		t.Context(), encodeCaller, &llm.ChatRequest{Messages: []*llm.ChatMessage{{Text: "\xff"}}}, connection.PriorityBackground,
	); err == nil || !strings.Contains(err.Error(), "llm: encode "+llm.OperationChat) {
		t.Errorf("Chat invalid UTF-8 err = %v", err)
	}
	if len(encodeCaller.operations) != 0 {
		t.Errorf("expected no calls when request marshal fails, got %v", encodeCaller.operations)
	}

	caller := newCaller(llm.Roles{})

	if _, err := llm.Complete(t.Context(), caller, &llm.CompleteRequest{}, connection.PriorityBackground); !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("Complete err = %v, want ErrNoConnection", err)
	}
	if _, err := llm.Embed(t.Context(), caller, &llm.EmbedRequest{Texts: []string{"hi"}}, connection.PriorityBackground); !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("Embed err = %v, want ErrNoConnection", err)
	}
	if _, err := llm.GetProfile(t.Context(), caller); !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("GetProfile err = %v, want ErrNoConnection", err)
	}

	callErr := errors.New("call failed")
	errCaller := &staticCaller{err: callErr}
	if _, err := llm.Complete(t.Context(), errCaller, &llm.CompleteRequest{}, connection.PriorityBackground); !errors.Is(err, callErr) {
		t.Errorf("Complete callErr = %v, want %v", err, callErr)
	}
	if _, err := llm.Chat(t.Context(), errCaller, &llm.ChatRequest{}, connection.PriorityBackground); !errors.Is(err, callErr) {
		t.Errorf("Chat callErr = %v, want %v", err, callErr)
	}

	badDoneCaller := &staticCaller{frames: &frames{done: []byte{0xff, 0xff}}}
	if _, err := llm.Complete(t.Context(), badDoneCaller, &llm.CompleteRequest{}, connection.PriorityBackground); err == nil {
		t.Error("expected error for invalid Complete done bytes")
	}

	validFrameBytes, err := proto.Marshal(&llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: "ok"}})
	if err != nil {
		t.Fatal(err)
	}
	validDoneBytes, err := proto.Marshal(&llm.ChatFinish{Reason: llm.FinishReason_FINISH_REASON_STOP})
	if err != nil {
		t.Fatal(err)
	}

	badFrameStream, err := llm.Chat(t.Context(), &staticCaller{frames: &frames{
		data: [][]byte{{0xff, 0xff}, validFrameBytes},
		done: validDoneBytes,
	}}, &llm.ChatRequest{}, connection.PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	firstBad := badFrameStream.Next()
	secondBad := badFrameStream.Next()
	if firstBad || secondBad {
		t.Error("expected Next to return false after invalid frame")
	}
	if badFrameStream.Err() == nil || badFrameStream.Finish() != nil {
		t.Errorf("badFrameStream Err = %v, Finish = %v", badFrameStream.Err(), badFrameStream.Finish())
	}

	errStream, err := llm.Chat(t.Context(), &staticCaller{frames: &frames{
		done: validDoneBytes,
		err:  callErr,
	}}, &llm.ChatRequest{}, connection.PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	if errStream.Next() || errStream.Finish() != nil || !errors.Is(errStream.Err(), callErr) {
		t.Errorf("errStream Finish = %v, Err = %v", errStream.Finish(), errStream.Err())
	}

	badFinishStream, err := llm.Chat(t.Context(), &staticCaller{frames: &frames{
		done: []byte{0xff, 0xff},
	}}, &llm.ChatRequest{}, connection.PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	if badFinishStream.Next() || badFinishStream.Finish() != nil || badFinishStream.Err() == nil {
		t.Errorf("badFinishStream Finish = %v, Err = %v", badFinishStream.Finish(), badFinishStream.Err())
	}
}

func TestAPayloadThatIsNoRequestOfTheContractIsAnInvalidRequest(t *testing.T) {
	provider := llm.Provide(nil, llm.Roles{Completer: &assistant{}, Chatter: &assistant{}})

	for _, op := range []string{llm.OperationComplete, llm.OperationChat} {
		_, err := provider[op](t.Context(), connection.Call{}, []byte{0xff, 0xff}, func([]byte) error { return nil })
		if !connection.IsStop(err) || err.Error() != connection.ReasonInvalidRequest.String() {
			t.Errorf("%s err = %v, want a stop of an invalid request", op, err)
		}
	}

	roleErr := errors.New("role failed")
	failServe := llm.Provide(nil, llm.Roles{Completer: &failingAssistant{err: roleErr}})[llm.OperationComplete]
	if _, err := failServe(t.Context(), connection.Call{}, nil, func([]byte) error { return nil }); !errors.Is(err, roleErr) {
		t.Errorf("failServe err = %v, want %v", err, roleErr)
	}

	sendErr := errors.New("send failed")
	if _, err := provider[llm.OperationChat](t.Context(), connection.Call{}, nil, func([]byte) error { return sendErr }); !errors.Is(err, sendErr) {
		t.Errorf("chat sendErr = %v, want %v", err, sendErr)
	}

	sent := false
	badFrameServe := llm.Provide(nil, llm.Roles{
		Chatter: &failingAssistant{frame: &llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: "\xff"}}},
	})[llm.OperationChat]
	if _, err := badFrameServe(t.Context(), connection.Call{}, nil, func([]byte) error {
		sent = true
		return nil
	}); err == nil || sent {
		t.Errorf("badFrameServe err = %v, sent = %v, want marshal error without calling send", err, sent)
	}
}

type protoMessage interface {
	proto.Message
	Reset()
	String() string
	Descriptor() ([]byte, []int)
}

func assertProtoMessage(t *testing.T, msg protoMessage) {
	t.Helper()
	if msg.ProtoReflect() == nil || msg.ProtoReflect().Descriptor() == nil {
		t.Fatalf("ProtoReflect nil for %T", msg)
	}
	if raw, idx := msg.Descriptor(); len(raw) == 0 || len(idx) == 0 {
		t.Fatalf("Descriptor empty for %T", msg)
	}
	if msg.String() == "" {
		t.Fatalf("String empty for populated %T", msg)
	}
	msg.Reset()
	if msg.String() != "" {
		t.Fatalf("String non-empty after Reset for %T: %q", msg, msg.String())
	}
}

func TestGeneratedProtoMethods(t *testing.T) {
	for _, v := range []llm.FinishReason{
		llm.FinishReason_FINISH_REASON_UNSPECIFIED,
		llm.FinishReason_FINISH_REASON_STOP,
		llm.FinishReason_FINISH_REASON_TOOL_CALLS,
		llm.FinishReason_FINISH_REASON_LENGTH,
	} {
		if got := v.Enum(); got == nil || *got != v {
			t.Fatalf("Enum(%v) = %v", v, got)
		}
		if v.String() == "" || v.Descriptor() == nil || v.Type() == nil || v.Number() != protoreflect.EnumNumber(v) {
			t.Fatalf("FinishReason methods failed for %v", v)
		}
		if raw, idx := v.EnumDescriptor(); len(raw) == 0 || len(idx) == 0 {
			t.Fatalf("EnumDescriptor(%v) = len %d, %v", v, len(raw), idx)
		}
	}

	temp := 0.5
	for _, msg := range []protoMessage{
		&llm.Message{Role: "user", Text: "hi"},
		&llm.CompleteRequest{Messages: []*llm.Message{{Role: "user"}}, Temperature: &temp, MaxTokens: 10},
		&llm.CompleteResult{Text: "ok"},
		&llm.ToolCall{Id: "1", Name: "n", ArgumentsJson: "{}"},
		&llm.ToolDefinition{Name: "n", Description: "d", ParametersJson: "{}"},
		&llm.ChatMessage{Role: "user", Text: "t", ToolCalls: []*llm.ToolCall{{Id: "1"}}, ToolCallId: "1"},
		&llm.ChatRequest{Messages: []*llm.ChatMessage{{Role: "user"}}, Tools: []*llm.ToolDefinition{{Name: "n"}}, Temperature: &temp, MaxTokens: 10},
		&llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: "d"}},
		&llm.ChatFinish{Reason: llm.FinishReason_FINISH_REASON_STOP},
		&llm.EmbedRequest{Texts: []string{"a"}},
		&llm.Vector{Values: []float32{1}},
		&llm.EmbedResult{Vectors: []*llm.Vector{{Values: []float32{1}}}},
		&llm.Profile{Model: "m", Source: "s", Pooling: "p", QueryPrefix: "q", DocumentPrefix: "d", Width: 1, MaxTokens: 2},
		&numenv1.RerankRequest{Query: "q", Instruction: "i", Documents: []string{"d"}},
		&numenv1.RerankResult{Scores: []float32{1}},
	} {
		assertProtoMessage(t, msg)
	}

	profReq := &llm.ProfileRequest{}
	if profReq.ProtoReflect() == nil || profReq.String() != "" {
		t.Fatalf("ProfileRequest = %v", profReq)
	}
	if raw, idx := profReq.Descriptor(); len(raw) == 0 || len(idx) == 0 {
		t.Fatalf("ProfileRequest.Descriptor = %d, %v", len(raw), idx)
	}
	profReq.Reset()

	var (
		nilMsg       *llm.Message
		nilCompReq   *llm.CompleteRequest
		nilCompRes   *llm.CompleteResult
		nilToolCall  *llm.ToolCall
		nilToolDef   *llm.ToolDefinition
		nilChatMsg   *llm.ChatMessage
		nilChatReq   *llm.ChatRequest
		nilChatFrame *llm.ChatFrame
		nilChatFin   *llm.ChatFinish
		nilEmbedReq  *llm.EmbedRequest
		nilVec       *llm.Vector
		nilEmbedRes  *llm.EmbedResult
		nilProfReq   *llm.ProfileRequest
		nilProf      *llm.Profile
		nilRerankReq *numenv1.RerankRequest
		nilRerankRes *numenv1.RerankResult
	)
	for _, m := range []proto.Message{
		nilMsg, nilCompReq, nilCompRes, nilToolCall, nilToolDef, nilChatMsg, nilChatReq,
		nilChatFrame, nilChatFin, nilEmbedReq, nilVec, nilEmbedRes, nilProfReq, nilProf, nilRerankReq, nilRerankRes,
	} {
		if m.ProtoReflect() == nil {
			t.Fatalf("nil ProtoReflect failed for %T", m)
		}
	}
}
