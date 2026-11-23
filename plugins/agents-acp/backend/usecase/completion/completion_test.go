package completion

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/settings"
)

// absentLauncher starts no program.
type absentLauncher struct{}

func (absentLauncher) Launch(context.Context, string, []string) (acp.Process, error) {
	return nil, errors.New("no program starts here")
}

type fakeListings struct{}

func (fakeListings) GetListing(context.Context, agents.Agent) acp.SessionModels {
	return acp.SessionModels{}
}

func TestRequestLimitIsTheSettingElseTwoMinutes(t *testing.T) {
	values := new(settings.Settings)
	subject, err := New(Dependencies{
		Settings: values, Launcher: absentLauncher{}, Listings: fakeListings{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := subject.getRequestLimit(); got != 120*time.Second {
		t.Errorf("default = %v, want 120s", got)
	}
	values.Apply(map[string]string{"requestTimeout": "45"})
	if got := subject.getRequestLimit(); got != 45*time.Second {
		t.Errorf("set to 45 = %v, want 45s", got)
	}
	values.Apply(map[string]string{"requestTimeout": "0.000000001"})
	if got := subject.getRequestLimit(); got != time.Nanosecond {
		t.Errorf("set to 1ns = %v, want 1ns", got)
	}
	values.Apply(map[string]string{"requestTimeout": "soon"})
	if got := subject.getRequestLimit(); got != 120*time.Second {
		t.Errorf("invalid value = %v, want 120s", got)
	}
}

func TestNewRefusesMissingDependencies(t *testing.T) {
	for _, deps := range []Dependencies{
		{},
		{Launcher: absentLauncher{}, Listings: fakeListings{}},
		{Settings: new(settings.Settings), Listings: fakeListings{}},
		{Settings: new(settings.Settings), Launcher: absentLauncher{}},
	} {
		if _, err := New(deps); err == nil {
			t.Errorf("New(%+v) accepted missing dependencies", deps)
		}
	}
}

func TestAProgramIsUnreachableWhenNoLookupIsGiven(t *testing.T) {
	subject, err := New(Dependencies{Settings: new(settings.Settings), Launcher: absentLauncher{}, Listings: fakeListings{}})
	if err != nil {
		t.Fatal(err)
	}
	emptyAgent, err := subject.getCallAgent(connection.Call{Type: "acp", Fields: map[string]string{"program": "  "}})
	if !connection.IsStop(err) || emptyAgent.Type != "acp" {
		t.Errorf("getCallAgent(empty) = (%+v, %v), want StopError and Type=acp", emptyAgent, err)
	}

	agent, err := subject.getCallAgent(connection.Call{Type: "acp", Fields: map[string]string{"program": "agent"}})
	retryAfter, isTryNext := connection.IsTryNext(err)
	if !isTryNext || retryAfter != 0 || agent.Type != "acp" {
		t.Errorf("getCallAgent = (%+v, %v, retryAfter=%v), want try next with 0 delay and Type=acp", agent, err, retryAfter)
	}
}

func TestCollectorEnforcesMaxAnswerBytes(t *testing.T) {
	var c collector
	if err := c.HandleUpdate(acp.Update{}); err != nil {
		t.Fatalf("HandleUpdate(empty) = %v, want nil", err)
	}
	chunk := strings.Repeat("a", 4<<20)
	if err := c.HandleUpdate(acp.Update{Chunk: &acp.Chunk{Text: chunk}}); err != nil {
		t.Fatalf("HandleUpdate(4 MiB) = %v, want nil", err)
	}
	if err := c.HandleUpdate(acp.Update{Chunk: &acp.Chunk{Text: "x"}}); !errors.Is(err, errAnswerTooLarge) {
		t.Fatalf("HandleUpdate(4 MiB + 1) = %v, want %v", err, errAnswerTooLarge)
	}
}

type stubACPProcess struct {
	out    chan []byte
	closed chan struct{}
	answer string
	hang   bool
}

func (p *stubACPProcess) Dir() string    { return "/tmp" }
func (p *stubACPProcess) Stderr() string { return "" }

func (p *stubACPProcess) Send(line []byte) error {
	var msg struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return err
	}
	switch msg.Method {
	case "initialize":
		p.out <- []byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":1,"agentCapabilities":{}}}`)
	case "session/new":
		p.out <- []byte(`{"jsonrpc":"2.0","id":2,"result":{"sessionId":"s1"}}`)
	case "session/prompt":
		if p.hang {
			return nil
		}
		if p.answer != "" {
			p.out <- []byte(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"` + p.answer + `"}}}}`)
		}
		p.out <- []byte(`{"jsonrpc":"2.0","id":3,"result":{"stopReason":"end_turn"}}`)
	}
	return nil
}

func (p *stubACPProcess) Receive() ([]byte, error) {
	select {
	case line := <-p.out:
		return line, nil
	case <-p.closed:
		return nil, io.EOF
	}
}

func (p *stubACPProcess) Close() error {
	select {
	case <-p.closed:
	default:
		close(p.closed)
	}
	return nil
}

type stubLauncher struct {
	answer string
	hang   bool
}

func (l stubLauncher) Launch(context.Context, string, []string) (acp.Process, error) {
	return &stubACPProcess{out: make(chan []byte, 8), closed: make(chan struct{}), answer: l.answer, hang: l.hang}, nil
}

func TestCompleteHandlesSuccessEmptyFailureAndTimeout(t *testing.T) {
	lookPath := func(name string) (string, error) { return "/bin/" + name, nil }
	call := connection.Call{Type: "acp", Fields: map[string]string{"program": "agent"}}
	req := &llm.CompleteRequest{Messages: []*llm.Message{{Role: "user", Text: "hello?"}}}

	okComp, err := New(Dependencies{
		Settings: new(settings.Settings),
		Launcher: stubLauncher{answer: "hello world"},
		Listings: fakeListings{},
		LookPath: lookPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := okComp.Complete(t.Context(), call, req)
	if err != nil || res.GetText() != "hello world" {
		t.Fatalf("Complete(success) = (%v, %v), want \"hello world\"", res, err)
	}

	emptyComp, err := New(Dependencies{
		Settings: new(settings.Settings),
		Launcher: stubLauncher{answer: "   "},
		Listings: fakeListings{},
		LookPath: lookPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emptyComp.Complete(t.Context(), call, req); !connection.IsStop(err) {
		t.Fatalf("Complete(empty) = %v, want StopError", err)
	}

	failComp, err := New(Dependencies{
		Settings: new(settings.Settings),
		Launcher: absentLauncher{},
		Listings: fakeListings{},
		LookPath: lookPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := failComp.Complete(t.Context(), call, req); !connection.IsStop(err) {
		t.Fatalf("Complete(fail) = %v, want StopError", err)
	}

	timeoutSettings := new(settings.Settings)
	timeoutSettings.Apply(map[string]string{"requestTimeout": "0.02"})
	timeoutComp, err := New(Dependencies{
		Settings: timeoutSettings,
		Launcher: stubLauncher{hang: true},
		Listings: fakeListings{},
		LookPath: lookPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = timeoutComp.Complete(t.Context(), call, req)
	if _, isTryNext := connection.IsTryNext(err); !isTryNext {
		t.Fatalf("Complete(timeout) = %v, want TryNextError", err)
	}
}
