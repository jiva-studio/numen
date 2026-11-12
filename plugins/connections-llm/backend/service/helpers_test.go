package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/internal/adapter/openaiapi"
)

const testKey = "sk-test-key-1234"

func startFakeServer(t *testing.T, config fakeopenai.Config) (*fakeopenai.Server, string) {
	t.Helper()
	fake := fakeopenai.New(config)
	ts := httptest.NewServer(fake)
	t.Cleanup(ts.Close)
	return fake, ts.URL + "/v1"
}

func newCall(typeID, address, key, model string) connection.Call {
	call := connection.Call{
		Type:    typeID,
		Fields:  map[string]string{"baseUrl": address},
		Secrets: map[string]connection.Secret{},
		Options: map[string]string{llm.OptionModel: model},
	}
	if key != "" {
		call.Secrets["apiKey"] = connection.Secret{Value: key}
	}
	return call
}

func newContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func oneUserMessage(text string) *llm.CompleteRequest {
	return &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: text}}}
}

type failureWant struct {
	isTryNext  bool
	reason     connection.Reason
	retryAfter time.Duration
}

func checkFailure(t *testing.T, err error, want failureWant) {
	t.Helper()
	if err == nil {
		t.Fatalf("got no error, want %+v", want)
	}
	retryAfter, isTryNext := connection.IsTryNext(err)
	if isTryNext != want.isTryNext || connection.IsStop(err) == want.isTryNext {
		t.Fatalf("err = %v: try-next %v, stop %v, want try-next %v", err, isTryNext, connection.IsStop(err), want.isTryNext)
	}
	reason, ok := connection.GetReason(err)
	if !ok || reason != want.reason {
		t.Errorf("reason = %v, want %v", reason, want.reason)
	}
	if retryAfter != want.retryAfter {
		t.Errorf("retryAfter = %v, want %v", retryAfter, want.retryAfter)
	}
}

// testService is the plugin built as Mount builds it, with the settings Configure delivers. Its calls go
// through the operations Mount serves, with the payloads and the frames encoded as on the wire.
type testService struct {
	built *service
}

func newTestService(t *testing.T) *testService {
	t.Helper()
	return newTestServiceWith(t, clock.System{})
}

func newTestServiceWith(t *testing.T, source openaiapi.Clock) *testService {
	t.Helper()
	built, err := newService(source)
	if err != nil {
		t.Fatal(err)
	}
	return &testService{built: built}
}

func (s *testService) Configure(values map[string]string) {
	s.built.limits.Configure(values)
}

func (s *testService) Complete(ctx context.Context, call connection.Call, request *llm.CompleteRequest) (*llm.CompleteResult, error) {
	return llmtest.Complete(ctx, s.built.provider.roles(), call, request)
}

func (s *testService) Chat(
	ctx context.Context, call connection.Call, request *llm.ChatRequest, send func(*llm.ChatFrame) error,
) (*llm.ChatFinish, error) {
	return llmtest.Chat(ctx, s.built.provider.roles(), call, request, send)
}

func (s *testService) Embed(ctx context.Context, call connection.Call, texts []string) ([][]float32, error) {
	result, err := llmtest.Embed(ctx, s.built.provider.roles(), call, &llm.EmbedRequest{Texts: texts})
	if err != nil {
		return nil, err
	}
	vectors := make([][]float32, len(result.GetVectors()))
	for i, one := range result.GetVectors() {
		vectors[i] = one.GetValues()
	}
	return vectors, nil
}

func (s *testService) GetProfile(ctx context.Context, call connection.Call) (*llm.Profile, error) {
	return llmtest.GetProfile(ctx, s.built.provider.roles(), call)
}

// ListModels and ListReasoningLevels answer numen.v1.Options, a message of the core protocol a plugin does
// not import, so they are asked of the use case the operation serves.
func (s *testService) ListModels(ctx context.Context, call connection.Call, operations ...string) ([]connection.Option, error) {
	return s.built.provider.models.ListModels(ctx, call, operations)
}

func (s *testService) ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.Option, error) {
	return s.built.provider.models.ListReasoningLevels(ctx, call)
}

func withReasoning(call connection.Call, level string) connection.Call {
	call.Options[llm.OptionReasoning] = level
	return call
}
