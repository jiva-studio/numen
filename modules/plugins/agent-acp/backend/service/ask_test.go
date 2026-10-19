package service

import (
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// askAgent sends the task and returns the text it said and why it stopped.
func askAgent(t *testing.T, svc *agentServer, task *numenv1.AskAgentRequest) (said, stopped string) {
	t.Helper()
	stream, err := agenttest.Serve(t, svc).AskAgent(t.Context(), connect.NewRequest(task))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
		said += stream.Msg().GetSaid()
		if reason := stream.Msg().GetStopped(); reason != "" {
			stopped = reason
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return said, stopped
}

func TestAskAgentSaysTheAgentsAnswerAndStopsDone(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	said, stopped := askAgent(t, serveAgents(agent), &numenv1.AskAgentRequest{
		Agent: "acp:fake", Instructions: "be brief", Prompt: "why?",
	})

	if said != "the answer" || stopped != reasonDone {
		t.Errorf("said %q, stopped %q", said, stopped)
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
	seen := readRecord(t, record)
	session := seen["session/new"][0]["params"].(map[string]any)
	if servers := session["mcpServers"].([]any); len(servers) != 0 {
		t.Errorf("the session was given MCP servers %v", servers)
	}

	other, otherRecord := fakeACPAgent(t, "other", "plain")
	said, stopped := askAgent(t, serveAgents(other), &numenv1.AskAgentRequest{
		Agent: "acp:other", Prompt: "why?", Tools: []string{"note_read"},
	})
	if said != "" || stopped != errNoTools.Error() {
		t.Errorf("said %q, stopped %q", said, stopped)
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

	said, _ := askAgent(t, &agentServer{}, &numenv1.AskAgentRequest{
		Agent: "acp:fake", Prompt: "why?", Command: agent.Command,
	})

	if said != "the answer" || len(readRecord(t, record)["session/prompt"]) != 1 {
		t.Errorf("said %q", said)
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
		said, stopped := askAgent(t, svc, task)
		if said != "" || stopped == "" || stopped == reasonDone {
			t.Errorf("%s: said %q, stopped %q", task.GetAgent(), said, stopped)
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
