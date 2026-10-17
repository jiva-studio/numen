package container_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/agentproofreader"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// makeConfig is a container carrying one profile under a name.
func makeConfig(name string, profile container.Profile) container.Config {
	cfg := container.DefaultProofreading()
	cfg.Profiles = map[string]container.Profile{name: profile}
	return container.Config{Proofreading: cfg}
}

type mockProofreadService struct{}

func (s *mockProofreadService) GetProofreaderInfo(
	_ context.Context,
	_ *connect.Request[v1.GetProofreaderInfoRequest],
) (*connect.Response[v1.GetProofreaderInfoResponse], error) {
	return connect.NewResponse(&v1.GetProofreaderInfoResponse{
		Name: "test-model",
	}), nil
}

func (s *mockProofreadService) Proofread(
	_ context.Context,
	req *connect.Request[v1.ProofreadRequest],
) (*connect.Response[v1.ProofreadResponse], error) {
	replies := make(map[int32]string)
	for _, b := range req.Msg.GetBatches() {
		replies[b.GetNumber()] = "fixed: " + b.GetPrompt()
	}
	return connect.NewResponse(&v1.ProofreadResponse{
		Replies: replies,
	}), nil
}

// A container nobody configured proofreads with nothing, and the machine's own
// settings are not read here.
func TestAContainerNobodyGaveSettingsProofreadsWithNothing(t *testing.T) {
	proofreader, why := container.Config{}.Proofreader("", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.GetName())
	}
}

// A name no profile carries is said. A person who named one and is silently
// given nothing has no way to find out that nothing is proofreading.
func TestAProfileNameNoProfileCarriesIsRefused(t *testing.T) {
	held := makeConfig("openrouter", container.ServiceDefaults())

	proofreader, why := held.Proofreader("agent", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "agent") {
		t.Errorf("the reason does not name the profile asked for: %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.GetName())
	}
}

// A profile that asks for a service without saying which model is refused, as a
// name nobody carries is.
func TestAServiceWithoutAModelNameIsRefused(t *testing.T) {
	service := container.ServiceDefaults()
	service.Name = ""

	_, why := makeConfig("openrouter", service).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "model") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
}

// A service profile without a plugin host or when plugin is absent degrades gracefully.
func TestAServiceProfileDegradesWhenPluginAbsent(t *testing.T) {
	service := container.ServiceDefaults()
	service.Name = "some-model"

	proofreader, why := makeConfig("openrouter", service).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("want error when plugin absent, got nil")
	}
	if !strings.Contains(why.Error(), "unavailable") {
		t.Errorf("expected unavailable error, got: %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.GetName())
	}
}

func TestTheProofreadingPluginWiredViaHostAnswersProofreader(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProofreadingServiceHandler(&mockProofreadService{}))
	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "proofreading")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.proofreading",
		"name": "test-model",
		"version": "1.0.0",
		"capabilities": ["proofreading"],
		"targets": ["desktop.editor"],
		"backend": {"type": "process", "entrypoint": "bin/proofreading"}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ plugin.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	cfg := makeConfig("openrouter", container.ServiceDefaults())
	cfg.Proofreading.Profiles["openrouter"] = container.Profile{
		Use:  container.UseService,
		Name: "test-model",
	}
	cfg.PluginHost = host

	proofreader, err := cfg.Proofreader("openrouter", proofread.ScanInstruction)
	if err != nil {
		t.Fatalf("Proofreader failed: %v", err)
	}
	if proofreader == nil {
		t.Fatal("no proofreader returned")
	}
	if got := proofreader.GetName(); got != "test-model" {
		t.Errorf("got name %q, want %q", got, "test-model")
	}
}

// fakeAgents is the plugins that run agent programs: it lists the programs it is given and
// writes down every question put to them.
type fakeAgents struct {
	programs []port.AgentProgram
	answer   string

	mu     sync.Mutex
	asked  []port.Question
	listed int
}

func (f *fakeAgents) StartRun(context.Context, port.Task) (port.Run, error) {
	return nil, errors.New("nothing here runs a task")
}

func (f *fakeAgents) EndConversation(context.Context, string) error { return nil }

func (f *fakeAgents) GetPrograms(context.Context, map[string][]string) ([]port.AgentProgram, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listed++
	return f.programs, nil
}

func (f *fakeAgents) Answer(_ context.Context, question port.Question) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, question)
	return f.answer, nil
}

func (f *fakeAgents) getAsked() []port.Question {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.asked)
}

func (f *fakeAgents) getListed() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.listed
}

// A profile reaching the program a person has installed asks the program the settings name, told the
// model and the command line of the profile and what the run is proofreading.
func TestAnAgentProfileAsksTheProgramTheSettingsName(t *testing.T) {
	profile := container.AgentDefaults()
	profile.Model = "haiku"
	profile.Command = []string{"/somewhere/claude"}

	agents := &fakeAgents{answer: "0|fixed"}
	held := makeConfig("agent", profile)
	held.Agents = agents
	held.Agent = settings.Agent{Use: settings.UseProgram("claude")}

	proofreader, why := held.Proofreader("agent", proofread.SpeechInstruction)
	if why != nil {
		t.Fatal(why)
	}
	if proofreader == nil {
		t.Fatal("no proofreader")
	}
	out, err := proofreader.Proofread(t.Context(), []domain.ProofreadBatch{{Number: 3, Prompt: "a line"}})
	if err != nil || out[3] != "0|fixed" {
		t.Fatalf("got %v, %v", out, err)
	}

	asked := agents.getAsked()
	if len(asked) != 1 {
		t.Fatalf("the program was asked %d times", len(asked))
	}
	if got := asked[0]; got.Program != "claude" || got.Model != "haiku" || got.Prompt != "a line" ||
		got.Instructions != proofread.SpeechInstruction || !slices.Equal(got.Command, []string{"/somewhere/claude"}) {
		t.Errorf("got %+v", got)
	}
	if agents.getListed() != 0 {
		t.Error("the programs were listed where the settings name one")
	}
}

// Where the settings leave the choice open, the question names no program, and the plugins choose.
func TestAnAgentProfileLeavesTheProgramToThePluginsWhereTheSettingsNameNone(t *testing.T) {
	profile := container.AgentDefaults()
	profile.Model = "haiku"

	agents := &fakeAgents{}
	held := makeConfig("agent", profile)
	held.Agents = agents

	proofreader, why := held.Proofreader("agent", proofread.ScanInstruction)
	if why != nil {
		t.Fatal(why)
	}
	if _, err := proofreader.Proofread(t.Context(), []domain.ProofreadBatch{{Number: 1, Prompt: "x"}}); err != nil {
		t.Fatal(err)
	}
	if asked := agents.getAsked(); len(asked) != 1 || asked[0].Program != "" {
		t.Errorf("asked %+v", asked)
	}
	if agents.getListed() != 0 {
		t.Error("the programs were listed to open a profile")
	}
}

// An installation that names no agent, or has no plugin to run one, names no profile that needs one.
func TestAnAgentProfileNoProgramCanAnswerIsRefused(t *testing.T) {
	profile := container.AgentDefaults()
	profile.Model = "haiku"

	namedNone := makeConfig("agent", profile)
	namedNone.Agents = &fakeAgents{}
	namedNone.Agent = settings.Agent{Use: settings.UseProgram("")}

	for name, held := range map[string]container.Config{
		"no plugins":             makeConfig("agent", profile),
		"the settings name none": namedNone,
	} {
		proofreader, why := held.Proofreader("agent", proofread.ScanInstruction)
		if why == nil {
			t.Errorf("%s: no reason", name)
		}
		if proofreader != nil {
			t.Errorf("%s: got a proofreader: %v", name, proofreader.GetName())
		}
	}
}

// A profile reached through neither a service nor the command line is said to
// be, and the profile is named. A person who wrote a word this application does
// not know is owed the news.
func TestAProfileReachedThroughNeitherIsRefused(t *testing.T) {
	for _, use := range []string{"", "telepathy"} {
		profile := container.Profile{Use: use, Name: "some-model", Model: "haiku"}
		held := makeConfig("mine", profile)
		held.Agents = &fakeAgents{}

		proofreader, why := held.Proofreader("mine", proofread.ScanInstruction)
		if why == nil {
			t.Fatalf("a profile used through %q was opened", use)
		}
		for _, word := range []string{"mine", container.UseService, container.UseAgent} {
			if !strings.Contains(why.Error(), word) {
				t.Errorf("the reason does not name %q: %v", word, why)
			}
		}
		if proofreader != nil {
			t.Errorf("got a proofreader: %v", proofreader.GetName())
		}
	}
}

// How much of a person's own model a run may take is the profile's to say, and it reaches the
// proofreader that asks the program.
func TestHowManyQuestionsStandAtOnceReachesTheProofreader(t *testing.T) {
	profile := container.AgentDefaults()
	profile.Model = "haiku"
	profile.InFlight = 3

	held := makeConfig("agent", profile)
	held.Agents = &fakeAgents{programs: []port.AgentProgram{{Name: "claude"}}}

	proofreader, why := held.Proofreader("agent", proofread.SpeechInstruction)
	if why != nil {
		t.Fatal(why)
	}
	asking, ok := proofreader.(*agentproofreader.Proofreader)
	if !ok {
		t.Fatalf("got %T", proofreader)
	}
	if asking.InFlight != 3 {
		t.Errorf("the proofreader was told %d questions may stand at once", asking.InFlight)
	}
}

// A queue asked for under no name is nothing asked for, and nothing is left.
func TestAQueueUnderNoNameLeavesNothing(t *testing.T) {
	service := container.ServiceDefaults()
	service.Name = "test-model"

	queue, why := makeConfig("openrouter", service).ProofreadQueue("", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// A queue under a name no profile carries is refused, as a proofreader is.
func TestAQueueUnderANameNoProfileCarriesIsRefused(t *testing.T) {
	held := makeConfig("openrouter", container.ServiceDefaults())

	queue, why := held.ProofreadQueue("agent", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "agent") {
		t.Errorf("the reason does not name the profile asked for: %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// A profile reaching the command line has no queue: it is asked and answers.
func TestAnAgentProfileLeavesNothing(t *testing.T) {
	profile := container.AgentDefaults()
	profile.Model = "haiku"

	queue, why := makeConfig("agent", profile).ProofreadQueue("agent", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// The reason for a name nobody carries says that, and not that the profile is
// reached some other way.
func TestTheReasonForAMissingProfileSaysItIsMissing(t *testing.T) {
	held := makeConfig("openrouter", container.ServiceDefaults())

	for name, ask := range map[string]func() error{
		"proofreader": func() error { _, why := held.Proofreader("agent", proofread.ScanInstruction); return why },
		"queue":       func() error { _, why := held.ProofreadQueue("agent", proofread.ScanInstruction); return why },
	} {
		why := ask()
		if why == nil || !strings.Contains(why.Error(), "no proofreading profile named") {
			t.Errorf("the %s is refused with %v", name, why)
		}
	}
}

// A profile reaching the program that names no model is refused before the program is asked
// anything.
func TestAnAgentProfileWithoutAModelIsNotStarted(t *testing.T) {
	agents := &fakeAgents{}
	held := makeConfig("agent", container.AgentDefaults())
	held.Agents = agents

	_, why := held.Proofreader("agent", proofread.ScanInstruction)
	if why == nil || !strings.Contains(why.Error(), "names no model") {
		t.Errorf("the reason is %v", why)
	}
	if agents.getListed() != 0 || len(agents.getAsked()) != 0 {
		t.Error("the program was asked for a profile naming no model")
	}
}
