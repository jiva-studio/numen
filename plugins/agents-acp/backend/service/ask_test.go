package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

// askAgent sends the task and returns the response text and the stop reason.
func askAgent(t *testing.T, svc *agentServer, task *agentrun.AskAgentRequest) (said, stopReason string) {
	t.Helper()
	stream, err := agenttest.Serve(t, svc).AskAgent(t.Context(), connect.NewRequest(task))
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

	response, stopReason := askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{
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

	response, stopReason := askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsUrl: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
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

	askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsUrl: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
	})

	replies := readRecord(t, record)["reply"]
	if len(replies) < 4 {
		t.Fatalf("the agent had %d replies", len(replies))
	}
	allowed, denied := `{"outcome":{"optionId":"once","outcome":"selected"}}`, `{"outcome":{"outcome":"cancelled"}}`
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
	stream, err := agenttest.Serve(t, serveAgents(agent)).AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsUrl: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
		ToolDeclarations: []*agentrun.ToolDeclaration{{Name: "note_read", Title: "Read a note", Kind: "read", About: "path"}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	var steps []string
	for stream.Receive() {
		switch step := stream.Msg().GetStep().(type) {
		case *agentrun.AskAgentResponse_ToolCall:
			call := step.ToolCall
			steps = append(steps, fmt.Sprintf("call %s %s %s %s %s", call.GetCall(), call.GetTool(), call.GetKind(), call.GetAbout(), call.GetPath()))
		case *agentrun.AskAgentResponse_ToolFinished:
			steps = append(steps, "finished")
		case *agentrun.AskAgentResponse_Text:
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

	askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{
		Agent: "acp:fake", Prompt: "why?", ToolsUrl: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
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

	response, stopReason := askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{
		Agent: "acp:fake", Prompt: "why?", Tools: []string{"note_read"},
		ToolsUrl: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
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

	askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{Agent: "acp:fake", Model: "m2", Prompt: "why?"})

	set := readRecord(t, record)["session/set_config_option"]
	if len(set) != 1 || getField[string](t, getField[map[string]any](t, set[0], "params"), "value") != "m2" {
		t.Errorf("the agent was sent %v, want m2", set)
	}
}

// A task carries the command line of its program, which starts the agent where the settings gave
// none.
func TestAskAgentStartsTheAgentByTheCommandOfTheTask(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	response, _ := askAgent(t, &agentServer{}, &agentrun.AskAgentRequest{
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

	for _, task := range []*agentrun.AskAgentRequest{
		{Agent: "acp:failing", Prompt: "why?"},
		{Agent: "acp:refusing", Prompt: "why?"},
		{Agent: "acp:absent", Prompt: "why?"},
		{Agent: "claude", Prompt: "why?"},
	} {
		response, stopReason := askAgent(t, svc, task)
		if response != "" || stopReason == "" || stopReason == reasonDone {
			t.Errorf("%s: said %q, stopReason %q", task.GetAgent(), response, stopReason)
		}
	}
}

// Each agent the settings name is a program, listed by its prefixed name with the models its
// session reports, the one it answers with the default.
func TestTheProgramsAreTheACPAgentsTheSettingsName(t *testing.T) {
	config, _ := fakeACPAgent(t, "config", "config")
	plain, _ := fakeACPAgent(t, "plain", "plain")
	svc := &agentServer{}

	got, err := svc.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{
		Commands: []*agentrun.AgentProgramCommand{
			{Program: "acp:plain", Command: plain.Command},
			{Program: "acp:config", Command: config.Command},
			{Program: "other", Command: []string{"/bin/other"}},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	programs := got.Msg.GetPrograms()
	if len(programs) != 2 || programs[0].GetName() != "acp:config" || programs[1].GetName() != "acp:plain" {
		t.Fatalf("the programs are %v", programs)
	}
	if !programs[0].GetIsInstalled() || !programs[0].GetLimitsTools() || programs[0].GetIsDefault() {
		t.Errorf("the program is %v", programs[0])
	}
	models := programs[0].GetModels()
	if len(models) != 2 || models[0].GetName() != "m1" || models[0].GetTitle() != "Model One" || !models[0].GetIsDefault() ||
		models[1].GetName() != "m2" || models[1].GetIsDefault() {
		t.Errorf("the models are %v", models)
	}
	if len(programs[1].GetModels()) != 0 {
		t.Errorf("the models are %v", programs[1].GetModels())
	}
}

// The session has no MCP servers, the client offers no file system or terminal, and every request
func TestAskAgent_RejectsEveryAgentRequest(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{Agent: "acp:fake", Prompt: "why?"})

	observed := readRecord(t, record)
	session := getField[map[string]any](t, observed["session/new"][0], "params")
	if cwd, ok := session["cwd"].(string); !ok || cwd == "" || !filepath.IsAbs(cwd) {
		t.Errorf("the session was opened in %q", cwd)
	}
	capabilities := getField[map[string]any](t, getField[map[string]any](t, observed["initialize"][0], "params"), "clientCapabilities")
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
	if outcome := getField[map[string]any](t, getField[map[string]any](t, replies[0], "result"), "outcome")["outcome"]; outcome != "cancelled" {
		t.Errorf("the permission was answered %v, want cancelled", outcome)
	}
	if replies[1]["error"] == nil || replies[1]["result"] != nil {
		t.Errorf("the file read was answered %v, want an error", replies[1])
	}
	if len(observed["session/close"]) != 1 {
		t.Errorf("the session was closed %d times, want 1", len(observed["session/close"]))
	}
}

func TestAskAgent_SelectsModelFromLegacyModelList(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "legacy")

	askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{Agent: "acp:fake", Model: "m2", Prompt: "why?"})

	set := readRecord(t, record)["session/set_model"]
	if len(set) != 1 || getField[string](t, getField[map[string]any](t, set[0], "params"), "modelId") != "m2" {
		t.Errorf("the agent was sent %v, want m2", set)
	}
}

// A model the agent does not offer stops the task before the agent is prompted.
func TestAskAgent_FailsOnUnavailableModel(t *testing.T) {
	for program, behaviour := range map[string]string{"fake": "config", "plain": "plain"} {
		agent, record := fakeACPAgent(t, program, behaviour)

		response, stopReason := askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{
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

	_, stopReason := askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{Agent: "acp:fake", Prompt: "why?"})

	if !strings.Contains(stopReason, "not signed in") {
		t.Errorf("stopped with %q, want the agent's own message", stopReason)
	}
}

func TestAskAgentStopsTheAgentWhenCancelled(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "hang")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	stream, err := agenttest.Serve(t, serveAgents(agent)).AskAgent(ctx, connect.NewRequest(
		&agentrun.AskAgentRequest{Agent: "acp:fake", Prompt: "why?"}))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
		if stream.Msg().GetStopped() != "" || stream.Msg().GetText() != "" {
			t.Errorf("a cancelled task said %v", stream.Msg())
		}
	}
	if connect.CodeOf(stream.Err()) != connect.CodeDeadlineExceeded && connect.CodeOf(stream.Err()) != connect.CodeCanceled {
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
		entry := program.GetName() + ":"
		for _, one := range program.GetModels() {
			entry += " " + one.GetName() + "=" + one.GetTitle()
			if one.GetIsDefault() {
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

func listPrograms(t *testing.T, svc *agentServer) []*agentrun.AgentProgram {
	t.Helper()
	res, err := svc.GetAgentPrograms(t.Context(), connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Msg.GetPrograms()
}

func TestGetAgentPrograms_ModelCache(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "config")
	svc := serveAgents(agent)

	first := listPrograms(t, svc)
	second := listPrograms(t, svc)

	if len(first) != 1 || len(first[0].GetModels()) != 2 || len(second) != 1 || len(second[0].GetModels()) != 2 {
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
	agent, record := fakeACPAgent(t, "failing", "fail")
	now := time.Unix(1000, 0)
	svc := serveAgents(agent)
	svc.getModelCache().SetClock(func() time.Time { return now })

	listPrograms(t, svc)
	listPrograms(t, svc)
	if started := readRecord(t, record)["started"]; len(started) != 1 {
		t.Fatalf("the agent was started %d times within the backoff, want 1", len(started))
	}
	now = now.Add(acpFailureBackoff)
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

	got, err := agenttest.Serve(t, serveAgents(agent)).GetAgentPrograms(t.Context(),
		connect.NewRequest(&agentrun.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range got.Msg.GetPrograms()[0].GetModels() {
		want := 0
		if one.GetName() == "m1" {
			want = 3
		}
		if len(one.GetReasoningLevels()) != want || (want > 0 && one.GetDefaultReasoning() != "medium") {
			t.Errorf("the model %v offers the wrong levels", one)
		}
	}
}

func TestAskAgentSetsTheDepthTheTaskNamesWhereTheAgentReportsIt(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "thought")

	askAgent(t, serveAgents(agent), &agentrun.AskAgentRequest{
		Agent: "acp:fake", Reasoning: "high", Prompt: "why?",
	})

	set := readRecord(t, record)["session/set_config_option"]
	if len(set) != 1 || getField[string](t, getField[map[string]any](t, set[0], "params"), "configId") != "effort" ||
		getField[string](t, getField[map[string]any](t, set[0], "params"), "value") != "high" {
		t.Errorf("the agent was sent %v, want high", set)
	}

	other, otherRecord := fakeACPAgent(t, "other", "thought")
	askAgent(t, serveAgents(other), &agentrun.AskAgentRequest{Agent: "acp:other", Reasoning: "max", Prompt: "why?"})
	if set := readRecord(t, otherRecord)["session/set_config_option"]; len(set) != 0 {
		t.Errorf("a depth the agent does not report was sent: %v", set)
	}
}
