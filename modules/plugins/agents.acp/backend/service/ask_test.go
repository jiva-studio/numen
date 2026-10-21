package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// askAgent sends the task and returns the text it said and why it stopped.
func askAgent(t *testing.T, svc *agentServer, task *numenv1.AskAgentRequest) (said, stopReason string) {
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

	response, stopReason := askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{
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
	for _, block := range prompts[0]["params"].(map[string]any)["prompt"].([]any) {
		texts = append(texts, block.(map[string]any)["text"].(string))
	}
	if strings.Join(texts, "|") != "be brief|why?" {
		t.Errorf("the prompt held %q, want the instructions and then the question", texts)
	}
}

// A task offers the agent no MCP server, whatever endpoint it holds, and a task naming a tool is
// stopped before the agent is started.
func TestAskAgentOffersTheAgentNoTool(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")
	svc := serveAgents(agent)

	askAgent(t, svc, &numenv1.AskAgentRequest{
		Agent: "acp:fake", Prompt: "why?", ToolsUrl: "http://127.0.0.1:1/mcp", ToolsToken: "secret",
	})
	observed := readRecord(t, record)
	session := observed["session/new"][0]["params"].(map[string]any)
	if servers := session["mcpServers"].([]any); len(servers) != 0 {
		t.Errorf("the session was given MCP servers %v", servers)
	}

	other, otherRecord := fakeACPAgent(t, "other", "plain")
	response, stopReason := askAgent(t, serveAgents(other), &numenv1.AskAgentRequest{
		Agent: "acp:other", Prompt: "why?", Tools: []string{"note_read"},
	})
	if response != "" || stopReason != errNoTools.Error() {
		t.Errorf("said %q, stopReason %q", response, stopReason)
	}
	if _, err := os.Stat(otherRecord); err == nil {
		t.Error("the agent was started for a task naming a tool")
	}
}

func TestAskAgentChoosesTheModelTheTaskNames(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "config")

	askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{Agent: "acp:fake", Model: "m2", Prompt: "why?"})

	set := readRecord(t, record)["session/set_config_option"]
	if len(set) != 1 || set[0]["params"].(map[string]any)["value"] != "m2" {
		t.Errorf("the agent was sent %v, want m2", set)
	}
}

// A task carries the command line of its program, which starts the agent where the settings gave
// none.
func TestAskAgentStartsTheAgentByTheCommandOfTheTask(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	response, _ := askAgent(t, &agentServer{}, &numenv1.AskAgentRequest{
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

	for _, task := range []*numenv1.AskAgentRequest{
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

	got, err := svc.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{
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
// of the agent is refused.
func TestAskAgentRefusesEveryRequestOfTheAgent(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{Agent: "acp:fake", Prompt: "why?"})

	observed := readRecord(t, record)
	session := observed["session/new"][0]["params"].(map[string]any)
	if cwd, _ := session["cwd"].(string); cwd == "" || !filepath.IsAbs(cwd) {
		t.Errorf("the session was opened in %q", cwd)
	}
	capabilities := observed["initialize"][0]["params"].(map[string]any)["clientCapabilities"].(map[string]any)
	if capabilities["terminal"] != false {
		t.Errorf("the client offered a terminal: %v", capabilities)
	}
	for name, proposed := range capabilities["fs"].(map[string]any) {
		if proposed != false {
			t.Errorf("the client offered %s", name)
		}
	}
	replies := observed["reply"]
	if len(replies) != 2 {
		t.Fatalf("the agent had %d replies, want 2", len(replies))
	}
	if outcome := replies[0]["result"].(map[string]any)["outcome"].(map[string]any)["outcome"]; outcome != "cancelled" {
		t.Errorf("the permission was answered %v, want cancelled", outcome)
	}
	if replies[1]["error"] == nil || replies[1]["result"] != nil {
		t.Errorf("the file read was answered %v, want an error", replies[1])
	}
	if len(observed["session/close"]) != 1 {
		t.Errorf("the session was closed %d times, want 1", len(observed["session/close"]))
	}
}

func TestAskAgentChoosesTheModelTheAgentOffersTheOlderWay(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "legacy")

	askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{Agent: "acp:fake", Model: "m2", Prompt: "why?"})

	set := readRecord(t, record)["session/set_model"]
	if len(set) != 1 || set[0]["params"].(map[string]any)["modelId"] != "m2" {
		t.Errorf("the agent was sent %v, want m2", set)
	}
}

// A model the agent does not offer stops the task before the agent is prompted.
func TestAskAgentStopsOnAModelTheAgentDoesNotOffer(t *testing.T) {
	for program, behaviour := range map[string]string{"fake": "config", "plain": "plain"} {
		agent, record := fakeACPAgent(t, program, behaviour)

		response, stopReason := askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{
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

	_, stopReason := askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{Agent: "acp:fake", Prompt: "why?"})

	if !strings.Contains(stopReason, "not signed in") {
		t.Errorf("stopped with %q, want the agent's own message", stopReason)
	}
}

func TestAskAgentStopsTheAgentWhenCancelled(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "hang")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	stream, err := agenttest.Serve(t, serveAgents(agent)).AskAgent(ctx, connect.NewRequest(
		&numenv1.AskAgentRequest{Agent: "acp:fake", Prompt: "why?"}))
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

// Each agent lists the models its session reports, the one it answers with the default; an agent
// that reports none, or cannot be asked, is listed with none.
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

func listPrograms(t *testing.T, svc *agentServer) []*numenv1.AgentProgram {
	t.Helper()
	res, err := svc.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
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

// An agent that failed to list is not asked again until the backoff has passed.
func TestAnAgentThatFailedToListIsLeftAlone(t *testing.T) {
	agent, record := fakeACPAgent(t, "failing", "fail")
	now := time.Unix(1000, 0)
	svc := serveAgents(agent)
	svc.acpModels.now = func() time.Time { return now }

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
func TestTheDepthsAnAgentReportsAreOfferedForItsCurrentModel(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "thought")

	got, err := agenttest.Serve(t, serveAgents(agent)).GetAgentPrograms(t.Context(),
		connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
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

	askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{
		Agent: "acp:fake", Reasoning: "high", Prompt: "why?",
	})

	set := readRecord(t, record)["session/set_config_option"]
	if len(set) != 1 || set[0]["params"].(map[string]any)["configId"] != "effort" ||
		set[0]["params"].(map[string]any)["value"] != "high" {
		t.Errorf("the agent was sent %v, want high", set)
	}

	other, otherRecord := fakeACPAgent(t, "other", "thought")
	askAgent(t, serveAgents(other), &numenv1.AskAgentRequest{Agent: "acp:other", Reasoning: "max", Prompt: "why?"})
	if set := readRecord(t, otherRecord)["session/set_config_option"]; len(set) != 0 {
		t.Errorf("a depth the agent does not report was sent: %v", set)
	}
}
