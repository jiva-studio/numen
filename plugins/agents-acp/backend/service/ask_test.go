package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
)

func askAgent(t *testing.T, svc *agentServer, task *agentrun.Request) (said, stopReason string) {
	t.Helper()
	stream, err := agenttest.Serve(t, svc).Ask(t.Context(), task)
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
		said += stream.Msg().GetText()
		if reason := stream.Msg().GetStopped(); reason != "" {
			stopReason = reason
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return said, stopReason
}

func TestAskAgent_ResponseSuccess(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	response, stopReason := askAgent(t, serveAgents(agent), &agentrun.Request{
		Agent: "acp:fake", Instructions: "be brief", Prompt: "why?",
	})

	if response != "the answer" || stopReason != reasonDone {
		t.Errorf("said %q, stopReason %q", response, stopReason)
	}
	prompts := readRecord(t, record)["session/prompt"]
	if len(prompts) != 1 {
		t.Fatalf("the agent was prompted %d times, want 1", len(prompts))
	}
	var texts []string
	for _, block := range getField[[]any](t, getField[map[string]any](t, prompts[0], "params"), "prompt") {
		texts = append(texts, getField[string](t, block, "text"))
	}
	if strings.Join(texts, "|") != "be brief|why?" {
		t.Errorf("the prompt held %q, want the instructions and then the question", texts)
	}
}

// A task naming tools opens the session with the endpoint of the run as its MCP server, and the
// agent is allowed once to call a tool of the task and nothing else.
func TestAskAgentOpensTheSessionWithTheRunEndpoint(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "tools")

	response, stopReason := askAgent(t, serveAgents(agent), &agentrun.Request{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
	})

	if response != "the answer" || stopReason != reasonDone {
		t.Errorf("said %q, stopReason %q", response, stopReason)
	}
	observed := readRecord(t, record)
	servers, err := json.Marshal(getField[[]any](t, getField[map[string]any](t, observed["session/new"][0], "params"), "mcpServers"))
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"headers":[{"name":"Authorization","value":"Bearer secret"}],"name":"numen","type":"http","url":"http://127.0.0.1:1/mcp"}]`
	if string(servers) != want {
		t.Errorf("the session was given MCP servers %s, want %s", servers, want)
	}
}

// A permission is granted once to a call of a tool of the task, identified by the name ACP gives the call
// or, where the request names none, by the name the agent reported for that call. A title naming a
// tool of the task grants nothing.
func TestAskAgentAllowsTheToolsOfTheTaskByTheirNames(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "tools")

	askAgent(t, serveAgents(agent), &agentrun.Request{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
	})

	replies := readRecord(t, record)["reply"]
	if len(replies) < 4 {
		t.Fatalf("the agent had %d replies", len(replies))
	}
	allowed, denied := `{"outcome":{"optionId":"once","outcome":"selected"}}`, `{"outcome":{"outcome":"canceled"}}`
	for i, want := range []string{allowed, denied, denied, allowed} {
		outcome, err := json.Marshal(replies[i]["result"])
		if err != nil {
			t.Fatal(err)
		}
		if string(outcome) != want {
			t.Errorf("permission %d was answered %s, want %s", i, outcome, want)
		}
	}
}

// The tool calls the agent reports reach the stream as steps: the call, shown by the declaration
// of its tool, and its end once.
func TestAskAgentReportsTheToolCallsOfTheAgent(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "tools")
	stream, err := agenttest.Serve(t, serveAgents(agent)).Ask(t.Context(), &agentrun.Request{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
		ToolDeclarations: []toolcall.ToolDeclaration{{Name: "note_read", Title: "Read a note", Kind: "read", About: "path"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var steps []string
	for stream.Receive() {
		switch step := stream.Msg().GetStep().(type) {
		case *agentrun.ToolCallStep:
			call := step.ToolCall
			steps = append(steps, fmt.Sprintf("call %s %s %s %s %s", call.Call, call.Tool, call.Kind, call.About, call.Path))
		case *agentrun.ToolFinishedStep:
			steps = append(steps, "finished")
		case *agentrun.TextStep:
			steps = append(steps, "text")
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"call c0 Read a note read Garden.md Garden.md",
		"call c0 Read a note read Garden.md Garden.md",
		"finished",
		"text",
	}
	if !slices.Equal(steps, want) {
		t.Errorf("the steps were %q, want %q", steps, want)
	}
}

// A task that names no tool gives the session no MCP server, whatever endpoint it holds.
func TestAskAgentOpensASessionWithNoServerForATaskWithNoTool(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "tools")

	askAgent(t, serveAgents(agent), &agentrun.Request{
		Agent: "acp:fake", Prompt: "why?", ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
	})

	session := getField[map[string]any](t, readRecord(t, record)["session/new"][0], "params")
	if servers := getField[[]any](t, session, "mcpServers"); len(servers) != 0 {
		t.Errorf("the session was given MCP servers %v", servers)
	}
}

// A task naming tools is stopped by an agent that takes no MCP server over HTTP, before a session
// is opened.
func TestAskAgentStopsWhereTheAgentCannotReachTheTools(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	response, stopReason := askAgent(t, serveAgents(agent), &agentrun.Request{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsURL: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
	})

	if response != "" || !strings.Contains(stopReason, "takes no MCP server over HTTP") {
		t.Errorf("said %q, stopReason %q", response, stopReason)
	}
	if sessions := readRecord(t, record)["session/new"]; len(sessions) != 0 {
		t.Errorf("a session was opened: %v", sessions)
	}
}

func TestAskAgentChoosesTheModelTheTaskNames(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "config")

	askAgent(t, serveAgents(agent), &agentrun.Request{Agent: "acp:fake", Model: "m2", Prompt: "why?"})

	set := readRecord(t, record)["session/set_config_option"]
	if len(set) != 1 || getField[string](t, getField[map[string]any](t, set[0], "params"), "value") != "m2" {
		t.Errorf("the agent was sent %v, want m2", set)
	}
}

// A task carries the command line of its program, which starts the agent where the settings gave
// none.
func TestAskAgentStartsTheAgentByTheCommandOfTheTask(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	response, _ := askAgent(t, &agentServer{}, &agentrun.Request{
		Agent: "acp:fake", Prompt: "why?", Command: agent.Command,
	})

	if response != "the answer" || len(readRecord(t, record)["session/prompt"]) != 1 {
		t.Errorf("said %q", response)
	}
}

func TestAskAgentStopsWithTheReasonItCannotAnswer(t *testing.T) {
	failing, _ := fakeACPAgent(t, "failing", "fail")
	refusing, _ := fakeACPAgent(t, "refusing", "refuse")
	svc := serveAgents(failing, refusing)

	for _, task := range []*agentrun.Request{
		{Agent: "acp:failing", Prompt: "why?"},
		{Agent: "acp:refusing", Prompt: "why?"},
		{Agent: "acp:absent", Prompt: "why?"},
		{Agent: "acp:", Command: []string{"/bin/true"}, Prompt: "why?"},
		{Agent: "claude", Prompt: "why?"},
	} {
		response, stopReason := askAgent(t, svc, task)
		if response != "" || stopReason == "" || stopReason == reasonDone {
			t.Errorf("%s: said %q, stopReason %q", task.Agent, response, stopReason)
		}
	}
	for _, task := range []*agentrun.Request{
		{Agent: "acp:absent", Prompt: "why?"},
		{Agent: "acp:", Command: []string{"/bin/true"}, Prompt: "why?"},
		{Agent: "claude", Prompt: "why?"},
	} {
		_, stopReason := askAgent(t, svc, task)
		if stopReason != errAgentNotInstalled.Error() {
			t.Errorf("%s: stopReason = %q, want %q", task.Agent, stopReason, errAgentNotInstalled.Error())
		}
	}
}

// Each agent the settings name is a program, listed by its prefixed name with the models its
// session reports, the one it answers with the default.
func TestTheProgramsAreTheACPAgentsTheSettingsName(t *testing.T) {
	config, _ := fakeACPAgent(t, "config", "config")
	plain, _ := fakeACPAgent(t, "plain", "plain")
	svc := &agentServer{}

	programs, err := svc.GetPrograms(t.Context(), []agentrun.ProgramCommand{
		{Program: "acp:plain", Command: plain.Command},
		{Program: "acp:config", Command: config.Command},
		{Program: "other", Command: []string{"/bin/other"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 2 || programs[0].Name != "acp:config" || programs[1].Name != "acp:plain" {
		t.Fatalf("the programs are %v", programs)
	}
	if !programs[0].IsInstalled || !programs[0].LimitsTools || programs[0].IsDefault {
		t.Errorf("the program is %v", programs[0])
	}
	models := programs[0].Models
	if len(models) != 2 || models[0].Name != "m1" || models[0].Title != "Model One" || !models[0].IsDefault ||
		models[1].Name != "m2" || models[1].IsDefault {
		t.Errorf("the models are %v", models)
	}
	if len(programs[1].Models) != 0 {
		t.Errorf("the models are %v", programs[1].Models)
	}
}

// The session has no MCP servers, the client offers no file system or terminal, and every request
func TestAskAgent_RejectsEveryAgentRequest(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	askAgent(t, serveAgents(agent), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"})

	observed := readRecord(t, record)
	initMsg := observed["initialize"][0]
	if getField[float64](t, initMsg, "id") != 1 {
		t.Errorf("initialize id = %v, want 1", initMsg["id"])
	}
	initParams := getField[map[string]any](t, initMsg, "params")
	if getField[float64](t, initParams, "protocolVersion") != 1 {
		t.Errorf("protocolVersion = %v, want 1", initParams["protocolVersion"])
	}
	session := getField[map[string]any](t, observed["session/new"][0], "params")
	if cwd, ok := session["cwd"].(string); !ok || cwd == "" || !filepath.IsAbs(cwd) {
		t.Errorf("the session was opened in %q", cwd)
	}
	capabilities := getField[map[string]any](t, initParams, "clientCapabilities")
	if capabilities["terminal"] != false {
		t.Errorf("the client offered a terminal: %v", capabilities)
	}
	for name, proposed := range getField[map[string]any](t, capabilities, "fs") {
		if proposed != false {
			t.Errorf("the client offered %s", name)
		}
	}
	replies := observed["reply"]
	if len(replies) != 2 {
		t.Fatalf("the agent had %d replies, want 2", len(replies))
	}
	if outcome := getField[map[string]any](t, getField[map[string]any](t, replies[0], "result"), "outcome")["outcome"]; outcome != "canceled" {
		t.Errorf("the permission was answered %v, want canceled", outcome)
	}
	if replies[1]["error"] == nil || replies[1]["result"] != nil {
		t.Errorf("the file read was answered %v, want an error", replies[1])
	}
	if code := getField[float64](t, getField[map[string]any](t, replies[1], "error"), "code"); code != -32601 {
		t.Errorf("error code = %v, want -32601", code)
	}
	if len(observed["session/close"]) != 1 {
		t.Errorf("the session was closed %d times, want 1", len(observed["session/close"]))
	}
}

func TestAskAgent_SelectsModelFromLegacyModelList(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "legacy")

	askAgent(t, serveAgents(agent), &agentrun.Request{Agent: "acp:fake", Model: "m2", Prompt: "why?"})

	set := readRecord(t, record)["session/set_model"]
	if len(set) != 1 || getField[string](t, getField[map[string]any](t, set[0], "params"), "modelId") != "m2" {
		t.Errorf("the agent was sent %v, want m2", set)
	}
}

// A model the agent does not offer stops the task before the agent is prompted.
func TestAskAgent_FailsOnUnavailableModel(t *testing.T) {
	for program, behavior := range map[string]string{"fake": "config", "plain": "plain"} {
		agent, record := fakeACPAgent(t, program, behavior)

		response, stopReason := askAgent(t, serveAgents(agent), &agentrun.Request{
			Agent: "acp:" + program, Model: "m9", Prompt: "why?",
		})

		if response != "" || stopReason == "" || stopReason == reasonDone {
			t.Errorf("%s: said %q, stopReason %q", program, response, stopReason)
		}
		if prompts := readRecord(t, record)["session/prompt"]; len(prompts) != 0 {
			t.Errorf("%s: the agent was prompted", program)
		}
	}
}

func TestAskAgent_FailureReason(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "fail")

	_, stopReason := askAgent(t, serveAgents(agent), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"})

	if !strings.Contains(stopReason, "not signed in") {
		t.Errorf("stopped with %q, want the agent's own message", stopReason)
	}
}

func TestAskAgentStopsTheAgentWhenCancelled(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "hang")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	stream, err := agenttest.Serve(t, serveAgents(agent)).Ask(ctx, &agentrun.Request{Agent: "acp:fake", Prompt: "why?"})
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
		if stream.Msg().GetStopped() != "" || stream.Msg().GetText() != "" {
			t.Errorf("a canceled task said %v", stream.Msg())
		}
	}
	if rpcerror.GetCode(stream.Err()) != rpcerror.DeadlineExceeded && rpcerror.GetCode(stream.Err()) != rpcerror.Canceled {
		t.Errorf("the stream ended with %v", stream.Err())
	}
}

// Each agent lists the models its session reports, with the one it responds with as the default; an agent
// that reports none, or cannot be queried, is listed with none.
func TestTheModelsOfAProgramAreThoseTheACPAgentReports(t *testing.T) {
	config, _ := fakeACPAgent(t, "config", "config")
	legacy, _ := fakeACPAgent(t, "legacy", "legacy")
	plain, _ := fakeACPAgent(t, "plain", "plain")
	failing, _ := fakeACPAgent(t, "failing", "fail")

	var got []string
	for _, program := range listPrograms(t, serveAgents(config, legacy, plain, failing)) {
		entry := program.Name + ":"
		for _, one := range program.Models {
			entry += " " + one.Name + "=" + one.Title
			if one.IsDefault {
				entry += "*"
			}
		}
		got = append(got, entry)
	}

	want := []string{
		"acp:config: m1=Model One* m2=Model Two",
		"acp:legacy: m1=Model One* m2=Model Two",
		"acp:plain:",
		"acp:failing:",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("listed\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func listPrograms(t *testing.T, svc *agentServer) []*agentrun.Program {
	t.Helper()
	res, err := svc.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestGetAgentPrograms_ModelCache(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "config")
	svc := serveAgents(agent)

	first := listPrograms(t, svc)
	second := listPrograms(t, svc)

	if len(first) != 1 || len(first[0].Models) != 2 || len(second) != 1 || len(second[0].Models) != 2 {
		t.Errorf("listed %v, then %v", first, second)
	}
	if started := readRecord(t, record)["initialize"]; len(started) != 1 {
		t.Errorf("the agent was started %d times, want 1", len(started))
	}
}

func TestNoProgramIsListedWithoutAnACPAgent(t *testing.T) {
	if got := listPrograms(t, &agentServer{}); len(got) != 0 {
		t.Errorf("listed %v", got)
	}
}

// An agent that failed to list is not queried again until the backoff has elapsed.
func TestAnAgentThatFailedToListIsLeftAlone(t *testing.T) {
	if acpListLimit != 15*time.Second || acpFailureBackoff != 30*time.Second {
		t.Fatalf("acpListLimit = %v, acpFailureBackoff = %v", acpListLimit, acpFailureBackoff)
	}
	agent, record := fakeACPAgent(t, "failing", "fail")
	now := clock.NewFake(time.Unix(1000, 0))
	svc := serveAgents(agent)
	svc.getModelCache().SetClock(now)

	listPrograms(t, svc)
	now.Advance(29 * time.Second)
	listPrograms(t, svc)
	if started := readRecord(t, record)["started"]; len(started) != 1 {
		t.Fatalf("the agent was started %d times within the backoff, want 1", len(started))
	}
	now.Advance(time.Second)
	listPrograms(t, svc)
	if started := readRecord(t, record)["started"]; len(started) != 2 {
		t.Errorf("the agent was started %d times after the backoff, want 2", len(started))
	}
}

func serveAgents(agents ...acpAgent) *agentServer {
	return &agentServer{acpAgents: func() []acpAgent { return agents }}
}

// The depths of the category thought_level an agent reports belong to the model of its session.
func TestGetAgentPrograms_ReturnsDepthsForCurrentModel(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "thought")

	got, err := agenttest.Serve(t, serveAgents(agent)).GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range got[0].Models {
		want := 0
		if one.Name == "m1" {
			want = 3
		}
		if len(one.ReasoningLevels) != want || (want > 0 && one.DefaultReasoning != "medium") {
			t.Errorf("the model %v offers the wrong levels", one)
		}
	}
}

func TestAskAgentSetsTheDepthTheTaskNamesWhereTheAgentReportsIt(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "thought")

	askAgent(t, serveAgents(agent), &agentrun.Request{
		Agent: "acp:fake", Reasoning: "high", Prompt: "why?",
	})

	set := readRecord(t, record)["session/set_config_option"]
	if len(set) != 1 || getField[string](t, getField[map[string]any](t, set[0], "params"), "configId") != "effort" ||
		getField[string](t, getField[map[string]any](t, set[0], "params"), "value") != "high" {
		t.Errorf("the agent was sent %v, want high", set)
	}

	other, otherRecord := fakeACPAgent(t, "other", "thought")
	askAgent(t, serveAgents(other), &agentrun.Request{Agent: "acp:other", Reasoning: "max", Prompt: "why?"})
	if set := readRecord(t, otherRecord)["session/set_config_option"]; len(set) != 0 {
		t.Errorf("a depth the agent does not report was sent: %v", set)
	}
}

func TestAskAndSessionEdgeCases(t *testing.T) {
	errSend := errors.New("send failed")
	agent, _ := fakeACPAgent(t, "fake", "plain")
	svc := serveAgents(agent)

	// Send failing on ModelStartedStep returns immediately.
	calls := 0
	err := svc.Ask(t.Context(), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(*agentrun.Response) error {
		calls++
		return errSend
	})
	if !errors.Is(err, errSend) || calls != 1 {
		t.Errorf("Ask on initial send error = %v (calls %d)", err, calls)
	}

	// Send failing on StoppedStep for missing agent returns the error.
	calls = 0
	err = svc.Ask(t.Context(), &agentrun.Request{Agent: "acp:absent", Prompt: "why?"}, func(res *agentrun.Response) error {
		calls++
		if res.GetStopped() != "" {
			return errSend
		}
		return nil
	})
	if !errors.Is(err, errSend) || calls != 2 {
		t.Errorf("Ask on stopped send error = %v (calls %d)", err, calls)
	}

	// Send failing on TextStep returns without sending StoppedStep.
	calls = 0
	err = svc.Ask(t.Context(), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(res *agentrun.Response) error {
		calls++
		if res.GetText() != "" {
			return errSend
		}
		return nil
	})
	if !errors.Is(err, errSend) || calls != 2 {
		t.Errorf("Ask on text send error = %v (calls %d)", err, calls)
	}

	// Canceled context returns ctx.Err() directly without sending StoppedStep.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var stopReason string
	err = svc.Ask(ctx, &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(res *agentrun.Response) error {
		if s := res.GetStopped(); s != "" {
			stopReason = s
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) || stopReason != "" {
		t.Errorf("Ask on canceled ctx = %v, stopReason = %q", err, stopReason)
	}

	// setReasoning failure and session close failure propagate from answer and readACPListing.
	failReasoning, _ := fakeACPAgent(t, "fail-reasoning", "fail-reasoning")
	if _, err := svc.answer(t.Context(), failReasoning, question{reasoning: "high", prompt: "why?"}); err == nil || !strings.Contains(err.Error(), "reasoning rejected") {
		t.Errorf("answer with failing setReasoning = %v", err)
	}
	failClose, _ := fakeACPAgent(t, "fail-close", "fail-close")
	if _, err := svc.answer(t.Context(), failClose, question{prompt: "why?"}); err == nil || !strings.Contains(err.Error(), "close rejected") {
		t.Errorf("answer with failing close = %v", err)
	}
	if _, err := readACPListing(t.Context(), failClose); err == nil || !strings.Contains(err.Error(), "close rejected") {
		t.Errorf("readACPListing with failing close = %v", err)
	}

	// Unnamed model in GetPrograms falls back to ID as Title.
	unnamedAgent := acpAgent{Program: "unnamed", Title: "Unnamed", Command: []string{"/bin/unnamed"}}
	unnamedSvc := serveAgents(unnamedAgent)
	unnamedSvc.getModelCache().Put("unnamed\x00/bin/unnamed", acpListing{
		Models:  []acpModel{{ID: "m-id", Name: ""}},
		Current: "m-id",
	})
	programs := listPrograms(t, unnamedSvc)
	if len(programs) != 1 || len(programs[0].Models) != 1 || programs[0].Models[0].Title != "m-id" {
		t.Errorf("unnamed model programs = %+v", programs)
	}

	// findAgent with single-element command succeeds.
	if found, err := svc.findAgent("acp:single", []string{"/bin/single"}); err != nil || found.Program != "single" || !slices.Equal(found.Command, []string{"/bin/single"}) {
		t.Errorf("findAgent single command = %+v, %v", found, err)
	}

	// openACPSession and start error paths and supervision.
	if _, err := openACPSession(t.Context(), acpAgent{Program: "empty", Command: nil}, toolEndpoint{}); err == nil || !strings.Contains(err.Error(), "no command starts empty") {
		t.Errorf("openACPSession empty command = %v", err)
	}
	if _, err := openACPSession(t.Context(), acpAgent{Program: "missing", Command: []string{"/nonexistent-acp-bin"}}, toolEndpoint{}); err == nil || !strings.Contains(err.Error(), "start missing") {
		t.Errorf("openACPSession missing binary = %v", err)
	}
	sess, err := openACPSession(t.Context(), agent, toolEndpoint{})
	if err != nil {
		t.Fatal(err)
	}
	if sess.cmd.Dir != sess.dir || sess.cmd.Cancel == nil || sess.cmd.WaitDelay <= 0 {
		t.Errorf("session cmd not supervised or dir mismatch: dir=%q want=%q waitDelay=%v", sess.cmd.Dir, sess.dir, sess.cmd.WaitDelay)
	}
	if closeErr := sess.close(); closeErr != nil {
		t.Errorf("close session: %v", closeErr)
	}

	stdinSet := &acpSession{cmd: exec.CommandContext(t.Context(), "/bin/true"), program: "stdin-prog"}
	stdinSet.cmd.Stdin = strings.NewReader("")
	if err := stdinSet.start(); err == nil || !strings.Contains(err.Error(), "open the input of stdin-prog") || errors.Unwrap(err) == nil {
		t.Errorf("start with Stdin set = %v", err)
	}
	stdoutSet := &acpSession{cmd: exec.CommandContext(t.Context(), "/bin/true"), program: "stdout-prog"}
	stdoutSet.cmd.Stdout = io.Discard
	if err := stdoutSet.start(); err == nil || !strings.Contains(err.Error(), "open the output of stdout-prog") || errors.Unwrap(err) == nil {
		t.Errorf("start with Stdout set = %v", err)
	}

	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "nonexistent-parent"))
	if _, err := openACPSession(t.Context(), agent, toolEndpoint{}); err == nil || !strings.Contains(err.Error(), "create the session folder") || errors.Unwrap(err) == nil {
		t.Errorf("openACPSession with invalid TMPDIR = %v", err)
	}

	// maxAnswerBytes and stderrTailBytes boundaries.
	chunkSess := &acpSession{id: "s1"}
	chunkSess.answer.WriteString(strings.Repeat("a", (4<<20)-4))
	validChunk := json.RawMessage(`{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"1234"}}}`)
	if err := chunkSess.readUpdate(validChunk); err != nil || chunkSess.answer.Len() != 4<<20 {
		t.Errorf("readUpdate at 4MiB boundary = %v (len %d)", err, chunkSess.answer.Len())
	}
	extraChunk := json.RawMessage(`{"sessionId":"s1","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"5"}}}`)
	if err := chunkSess.readUpdate(extraChunk); !errors.Is(err, errAnswerTooLarge) {
		t.Errorf("readUpdate past 4MiB boundary = %v, want errAnswerTooLarge", err)
	}

	var tb tailBuffer
	prefix := "dropped-"
	tail := strings.Repeat("k", 8<<10)
	if _, err := tb.Write([]byte(prefix + tail)); err != nil {
		t.Fatal(err)
	}
	if tb.String() != tail {
		t.Errorf("tailBuffer len = %d, want %d", len(tb.String()), len(tail))
	}
}
