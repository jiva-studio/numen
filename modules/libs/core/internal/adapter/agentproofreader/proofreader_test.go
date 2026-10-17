package agentproofreader_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/agentproofreader"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const (
	scanInstruction   = "proofread scan"
	speechInstruction = "proofread speech"
)

var aBatch = domain.ProofreadBatch{
	Number: 7,
	Prompt: "⟦0⟧the qulck brown fox ⟦1⟧jumped ovcr the lazy dog",
}

// answering is a program that answers with what a function says, and writes down every question.
type answering struct {
	reply func(ctx context.Context, question port.Question) (string, error)

	mu    sync.Mutex
	asked []port.Question
}

func (a *answering) Answer(ctx context.Context, question port.Question) (string, error) {
	a.mu.Lock()
	a.asked = append(a.asked, question)
	a.mu.Unlock()
	return a.reply(ctx, question)
}

func (a *answering) getAsked() []port.Question {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.asked)
}

func saying(text string) *answering {
	return &answering{reply: func(context.Context, port.Question) (string, error) { return text, nil }}
}

// Each batch is one question, put to the program with the command, the model and the instruction of the profile.
func TestABatchIsOneQuestionToTheProgram(t *testing.T) {
	program := saying("0|the quick brown fox")
	by := &agentproofreader.Proofreader{
		Answerer: program, Program: "claude", Command: []string{"/bin/claude", "--x"},
		Model: "haiku", Instruction: scanInstruction,
	}

	out, err := by.Proofread(t.Context(), []domain.ProofreadBatch{aBatch})
	if err != nil {
		t.Fatal(err)
	}
	if out[7] != "0|the quick brown fox" || len(out) != 1 {
		t.Errorf("the reply came back as %v", out)
	}

	want := port.Question{
		Program: "claude", Prompt: aBatch.Prompt, Instructions: scanInstruction,
		Model: "haiku", Command: []string{"/bin/claude", "--x"},
	}
	asked := program.getAsked()
	if len(asked) != 1 || asked[0].Program != want.Program || asked[0].Prompt != want.Prompt ||
		asked[0].Instructions != want.Instructions || asked[0].Model != want.Model ||
		!slices.Equal(asked[0].Command, want.Command) {
		t.Errorf("asked %+v, want %+v", asked, want)
	}
}

// The caller says what is being corrected: a scan and speech are corrected for different mistakes.
func TestTheInstructionIsTheCallersOwn(t *testing.T) {
	program := saying("")
	by := &agentproofreader.Proofreader{Answerer: program, Program: "claude", Instruction: speechInstruction}

	if _, err := by.Proofread(t.Context(), []domain.ProofreadBatch{aBatch}); err != nil {
		t.Fatal(err)
	}
	asked := program.getAsked()
	if asked[0].Instructions != speechInstruction || asked[0].Model != "" {
		t.Errorf("asked %+v", asked[0])
	}
}

// A batch nothing came back about is left out, and a run of nothing is nothing.
func TestABatchNothingCameBackAboutIsLeftOut(t *testing.T) {
	by := &agentproofreader.Proofreader{Answerer: saying(""), Program: "claude", Instruction: scanInstruction}

	out, err := by.Proofread(t.Context(), []domain.ProofreadBatch{aBatch})
	if err != nil || len(out) != 0 {
		t.Errorf("got %v, %v", out, err)
	}

	out, err = by.Proofread(t.Context(), nil)
	if err != nil || out != nil {
		t.Errorf("a run of no batches came back with %v, %v", out, err)
	}
}

// Without an instruction the program is not told what it is doing, and nothing is asked.
func TestNothingIsAskedWithoutAnInstruction(t *testing.T) {
	program := saying("x")
	by := &agentproofreader.Proofreader{Answerer: program, Program: "claude"}

	if _, err := by.Proofread(t.Context(), []domain.ProofreadBatch{aBatch}); err == nil {
		t.Error("a run with no instruction was taken")
	}
	if len(program.getAsked()) != 0 {
		t.Error("the program was asked without an instruction")
	}
}

// What put a correction right stands beside every line it made, and a model is what that is.
func TestTheNameSaysWhichModelCorrected(t *testing.T) {
	if got := (&agentproofreader.Proofreader{Program: "claude", Model: "haiku"}).GetName(); got != "claude haiku" {
		t.Errorf("got %q", got)
	}
	if got := (&agentproofreader.Proofreader{Program: "claude"}).GetName(); got != "claude" {
		t.Errorf("got %q", got)
	}
	if got := (&agentproofreader.Proofreader{Model: "haiku"}).GetName(); got != "agent haiku" {
		t.Errorf("got %q", got)
	}
}

// A program that failed is a run that failed: the batches already answered are dropped and the
// caller asks again.
func TestAProgramThatFailedEndsTheRun(t *testing.T) {
	program := &answering{reply: func(_ context.Context, question port.Question) (string, error) {
		if strings.Contains(question.Prompt, "bad") {
			return "", errors.New("no model")
		}
		return "fine", nil
	}}
	by := &agentproofreader.Proofreader{Answerer: program, Program: "claude", Instruction: scanInstruction, InFlight: 1}

	out, err := by.Proofread(t.Context(), []domain.ProofreadBatch{
		{Number: 1, Prompt: "good"}, {Number: 2, Prompt: "bad"}, {Number: 3, Prompt: "good"},
	})
	if err == nil {
		t.Fatal("no error")
	}
	if out != nil {
		t.Errorf("got %v", out)
	}
	if !strings.Contains(err.Error(), "no model") || !strings.Contains(err.Error(), "batch 2") {
		t.Errorf("the error does not say which batch failed and why: %v", err)
	}
}

// A run the person stopped ends when they stop it, and what it was answered before is not an answer.
func TestARunTheContextEndedEndsWithIt(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	program := &answering{reply: func(ctx context.Context, _ port.Question) (string, error) {
		once.Do(func() { close(started) })
		<-ctx.Done()
		return "0|late", ctx.Err()
	}}
	by := &agentproofreader.Proofreader{Answerer: program, Program: "claude", Instruction: scanInstruction}

	ctx, stop := context.WithCancel(t.Context())
	go func() {
		<-started
		stop()
	}()
	out, err := by.Proofread(ctx, []domain.ProofreadBatch{aBatch})
	if err == nil {
		t.Fatal("a run that was stopped came back with no reason")
	}
	if out != nil {
		t.Errorf("what a stopped run wrote was taken as an answer: %v", out)
	}
}

// A run that was stopped before it began asks nothing.
func TestARunStoppedBeforeItBeganAsksNothing(t *testing.T) {
	program := saying("x")
	by := &agentproofreader.Proofreader{Answerer: program, Program: "claude", Instruction: scanInstruction}

	ctx, stop := context.WithCancel(t.Context())
	stop()
	if _, err := by.Proofread(ctx, []domain.ProofreadBatch{aBatch}); err == nil {
		t.Error("a run that was stopped came back with no reason")
	}
}

// counting is a program that records how many questions stand at once. Each holds its turn until as
// many as the limit allows have arrived, so a limit that is too low shows as a peak below it.
type counting struct {
	reach int

	mu        sync.Mutex
	now, peak int
	gate      chan struct{}
	once      sync.Once
}

func newCounting(reach int) *counting {
	return &counting{reach: reach, gate: make(chan struct{})}
}

func (c *counting) Answer(ctx context.Context, _ port.Question) (string, error) {
	c.mu.Lock()
	c.now++
	c.peak = max(c.peak, c.now)
	if c.now >= c.reach {
		c.once.Do(func() { close(c.gate) })
	}
	c.mu.Unlock()

	select {
	case <-c.gate:
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
	}

	c.mu.Lock()
	c.now--
	c.mu.Unlock()
	return "", nil
}

func (c *counting) getPeak() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peak
}

func batches(n int) []domain.ProofreadBatch {
	out := make([]domain.ProofreadBatch, n)
	for i := range out {
		out[i] = domain.ProofreadBatch{Number: i, Prompt: "a line"}
	}
	return out
}

// A person's own model is left most of itself: no more questions stand at once than the profile
// allows.
func TestQuestionsNeverExceedTheInFlightLimit(t *testing.T) {
	for _, allowed := range []int{1, 2, 3} {
		program := newCounting(allowed)
		by := &agentproofreader.Proofreader{
			Answerer: program, Program: "claude", Instruction: "put it right", InFlight: allowed,
		}

		if _, err := by.Proofread(t.Context(), batches(6)); err != nil {
			t.Fatal(err)
		}
		if got := program.getPeak(); got != allowed {
			t.Errorf("%d allowed, %d stood at once", allowed, got)
		}
	}
}

// A profile naming none asks one at a time.
func TestAProfileNamingNoneAsksOneAtATime(t *testing.T) {
	program := newCounting(1)
	by := &agentproofreader.Proofreader{Answerer: program, Program: "claude", Instruction: "put it right"}

	if _, err := by.Proofread(t.Context(), batches(4)); err != nil {
		t.Fatal(err)
	}
	if got := program.getPeak(); got != 1 {
		t.Errorf("%d stood at once", got)
	}
}

// The limit is the proofreader's and not one run's: two callers asking at once take no more of a
// person's own model between them than the profile allows.
func TestTwoCallersAtOnceShareTheLimit(t *testing.T) {
	program := newCounting(2)
	by := &agentproofreader.Proofreader{
		Answerer: program, Program: "claude", Instruction: "put it right", InFlight: 2,
	}

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := by.Proofread(t.Context(), batches(3)); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	if got := program.getPeak(); got > 2 {
		t.Errorf("2 allowed, %d stood at once", got)
	}
}
