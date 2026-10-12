package container_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// makeConfig is a container carrying one profile under a name.
func makeConfig(name string, profile proofreading.Profile) container.Config {
	cfg := proofreading.Defaults()
	cfg.Profiles = map[string]proofreading.Profile{name: profile}
	return container.Config{Proofreading: cfg}
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
	held := makeConfig("openrouter", proofreading.ServiceDefaults())

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
	t.Setenv(proofreading.KeyEnvVar, "sk-test")

	_, why := makeConfig("openrouter", proofreading.ServiceDefaults()).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "model") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
}

func TestAModelWithoutAKeyGivesTheReasonAndNoProofreader(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "")
	service := proofreading.ServiceDefaults()
	service.Name = "some-model"

	proofreader, why := makeConfig("openrouter", service).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "key") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
	if proofreader != nil {
		t.Errorf("got a proofreader: %v", proofreader.GetName())
	}
}

func TestTheProofreadingSettingsGivenAreTheOnesUsed(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-test")
	service := proofreading.ServiceDefaults()
	service.Name = "some-model"
	// Nowhere: no test reaches a service.
	service.BaseURL = "http://127.0.0.1:1/v1"

	proofreader, why := makeConfig("openrouter", service).
		Proofreader("openrouter", proofread.ScanInstruction)
	if why != nil {
		t.Fatal(why)
	}
	if proofreader == nil {
		t.Fatal("no proofreader")
	}
	if got := proofreader.GetName(); got != "some-model" {
		t.Errorf("got %q", got)
	}
}

// A profile reaching the command line is opened by the platform, which is told
// what to start and what the run is proofreading.
func TestAnAgentProfileIsOpenedByThePlatform(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"
	profile.Command = []string{"/somewhere/claude"}

	var asked container.ProofreaderSpec
	held := makeConfig("agent", profile)
	held.AgentProofreader = func(said container.ProofreaderSpec) (port.Proofreader, error) {
		asked = said
		return spelling{}, nil
	}

	proofreader, why := held.Proofreader("agent", proofread.SpeechInstruction)
	if why != nil {
		t.Fatal(why)
	}
	if proofreader == nil {
		t.Fatal("no proofreader")
	}
	if asked.Model != "haiku" || len(asked.Command) != 1 || asked.Command[0] != "/somewhere/claude" {
		t.Errorf("got %+v", asked)
	}
	if asked.Instruction != proofread.SpeechInstruction {
		t.Errorf("the instruction is %q", asked.Instruction)
	}
}

// An installation that starts no process names no profile that needs one.
func TestAnAgentProfileWithoutAPlatformIsRefused(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"

	_, why := makeConfig("agent", profile).Proofreader("agent", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
}

// A proofreader configured with a queue is one the pages can be left with.
func TestAProofreaderWithAQueueLeavesPagesWithIt(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-a-key")
	service := proofreading.ServiceDefaults()
	service.Name = "test-model"

	queue, why := makeConfig("openrouter", service).
		ProofreadQueue("openrouter", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue == nil || queue.GetName() != "test-model" {
		t.Fatalf("got %v", queue)
	}
}

// A service with nowhere to leave the pages is asked a page at a time.
func TestAServiceWithoutAQueueLeavesNothing(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-a-key")
	service := proofreading.ServiceDefaults()
	service.Name = "test-model"
	service.BatchURL = ""

	queue, why := makeConfig("openrouter", service).
		ProofreadQueue("openrouter", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// A profile reaching the command line has no queue: it is asked and answers.
func TestAnAgentProfileLeavesNothing(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"

	queue, why := makeConfig("agent", profile).ProofreadQueue("agent", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// spelling is a proofreader that is never asked anything.
type spelling struct{}

func (spelling) GetName() string { return "a test" }

func (spelling) Proofread(_ context.Context, _ []proofread.Batch) (map[int]string, error) {
	return nil, errors.New("nothing here asks")
}

// A profile reached through neither a service nor the command line is said to
// be, and the profile is named. A person who wrote a word this application does
// not know is owed the news.
func TestAProfileReachedThroughNeitherIsRefused(t *testing.T) {
	for _, use := range []string{"", "telepathy"} {
		profile := proofreading.Profile{Use: use, Name: "some-model", Model: "haiku"}
		held := makeConfig("mine", profile)
		held.AgentProofreader = func(container.ProofreaderSpec) (port.Proofreader, error) {
			return spelling{}, nil
		}

		proofreader, why := held.Proofreader("mine", proofread.ScanInstruction)
		if why == nil {
			t.Fatalf("a profile used through %q was opened", use)
		}
		for _, word := range []string{"mine", proofreading.UseService, proofreading.UseAgent} {
			if !strings.Contains(why.Error(), word) {
				t.Errorf("the reason does not name %q: %v", word, why)
			}
		}
		if proofreader != nil {
			t.Errorf("got a proofreader: %v", proofreader.GetName())
		}
	}
}

// How much of a person's own model a run may take is the profile's to say, and
// it reaches the platform that starts the command line.
func TestHowManyRunsStandAtOnceReachesThePlatform(t *testing.T) {
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"
	profile.InFlight = 3

	var asked container.ProofreaderSpec
	held := makeConfig("agent", profile)
	held.AgentProofreader = func(said container.ProofreaderSpec) (port.Proofreader, error) {
		asked = said
		return spelling{}, nil
	}

	if _, why := held.Proofreader("agent", proofread.SpeechInstruction); why != nil {
		t.Fatal(why)
	}
	if asked.InFlight != 3 {
		t.Errorf("the platform was told %d runs may stand at once", asked.InFlight)
	}
}

// A queue asked for under no name is nothing asked for, and nothing is left.
func TestAQueueUnderNoNameLeavesNothing(t *testing.T) {
	service := proofreading.ServiceDefaults()
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
	held := makeConfig("openrouter", proofreading.ServiceDefaults())

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

// A queue for a service that names no model is refused.
func TestAQueueForAServiceWithoutAModelNameIsRefused(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-test")

	queue, why := makeConfig("openrouter", proofreading.ServiceDefaults()).
		ProofreadQueue("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "model") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// A queue that cannot be opened for want of a key says so and leaves nothing.
func TestAQueueWithoutAKeyGivesTheReasonAndNoQueue(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "")
	service := proofreading.ServiceDefaults()
	service.Name = "test-model"

	queue, why := makeConfig("openrouter", service).
		ProofreadQueue("openrouter", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if !strings.Contains(why.Error(), "key") {
		t.Errorf("the reason does not say what is missing: %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// A profile reaching the command line leaves nothing, whatever address it is
// given to leave pages at.
func TestAnAgentProfileWithABatchAddressLeavesNothing(t *testing.T) {
	t.Setenv(proofreading.KeyEnvVar, "sk-a-key")
	profile := proofreading.AgentDefaults()
	profile.Model = "haiku"
	profile.BatchURL = "http://127.0.0.1:1/v1/batches"

	queue, why := makeConfig("agent", profile).ProofreadQueue("agent", proofread.ScanInstruction)
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// A profile reached through neither a service nor the command line leaves no
// queue, and the reason is said.
func TestAProfileWithNeitherWayOfReachingItIsRefusedAsAQueue(t *testing.T) {
	profile := proofreading.Profile{Use: "telepathy", Name: "some-model", Model: "haiku"}

	queue, why := makeConfig("mine", profile).ProofreadQueue("mine", proofread.ScanInstruction)
	if why == nil {
		t.Fatal("no reason")
	}
	if queue != nil {
		t.Errorf("got a queue: %v", queue.GetName())
	}
}

// The reason for a name nobody carries says that, and not that the profile is
// reached some other way.
func TestTheReasonForAMissingProfileSaysItIsMissing(t *testing.T) {
	held := makeConfig("openrouter", proofreading.ServiceDefaults())

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

// A command line profile naming no model is refused before the platform is
// asked to start anything.
func TestAnAgentProfileWithoutAModelIsNotStarted(t *testing.T) {
	started := false
	held := makeConfig("agent", proofreading.AgentDefaults())
	held.AgentProofreader = func(container.ProofreaderSpec) (port.Proofreader, error) {
		started = true
		return spelling{}, nil
	}

	_, why := held.Proofreader("agent", proofread.ScanInstruction)
	if why == nil || !strings.Contains(why.Error(), "names no model") {
		t.Errorf("the reason is %v", why)
	}
	if started {
		t.Error("the platform was asked to start a profile naming no model")
	}
}
