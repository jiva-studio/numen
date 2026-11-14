package proofread

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

// model is a model whose reply is computed from the prompt by its script, and
// which records the instructions and prompts it received.
type model struct {
	script       func(prompt string) string
	instructions []string
	prompts      []string
	failure      error
}

func (m *model) ask(_ context.Context, instruction, prompt string) (string, error) {
	m.instructions = append(m.instructions, instruction)
	m.prompts = append(m.prompts, prompt)
	if m.failure != nil {
		return "", m.failure
	}
	return m.script(prompt), nil
}

func newReplyModel(reply string) *model {
	return &model{script: func(string) string { return reply }}
}

func twoPages() []numbered.Batch {
	return []numbered.Batch{
		{Number: 3, Lines: []numbered.Line{{Number: 10, Text: "the cta sat"}, {Number: 11, Text: "on the mat"}}},
		{Number: 4, Lines: []numbered.Line{{Number: 12, Text: "the dgo ran"}}},
	}
}

func TestProofreadScan_PromptsPerPageAndOrdersCorrections(t *testing.T) {
	m := &model{script: func(prompt string) string {
		if strings.Contains(prompt, "dgo") {
			return "12|the dog ran"
		}
		return "10|the cat sat"
	}}

	got, err := proofreadScan(t.Context(), m.ask, numbered.ScanInstruction, twoPages(), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []numbered.Line{{Number: 10, Last: 10, Text: "the cat sat"}, {Number: 12, Last: 12, Text: "the dog ran"}}
	if !reflect.DeepEqual(got.corrections, want) {
		t.Errorf("corrections %v, want %v", got.corrections, want)
	}
	if got.uncorrected != 0 || !got.isDone {
		t.Errorf("result %+v", got)
	}
	if len(m.prompts) != 2 || m.instructions[0] != numbered.ScanInstruction {
		t.Errorf("asked %d times under %q", len(m.prompts), m.instructions)
	}
}

func TestAScanPageWhoseReplyIsRefusedIsCountedUncorrected(t *testing.T) {
	got, err := proofreadScan(t.Context(), newReplyModel("999|invented").ask, numbered.ScanInstruction, twoPages(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.uncorrected != 2 || len(got.corrections) != 0 {
		t.Errorf("result %+v", got)
	}
}

func TestAScanFailureEndsTheRun(t *testing.T) {
	boom := errors.New("boom")
	ask := func(context.Context, string, string) (string, error) { return "", boom }
	if _, err := proofreadScan(t.Context(), ask, numbered.ScanInstruction, twoPages(), 0); !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestProofreadScan_AppliesEditLimit(t *testing.T) {
	pages := []numbered.Batch{{Number: 1, Lines: []numbered.Line{{Number: 1, Text: "abcdefghij"}}}}
	reply := newReplyModel("1|abcdeXYZ")

	got, err := proofreadScan(t.Context(), reply.ask, numbered.ScanInstruction, pages, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.corrections) != 0 {
		t.Errorf("default limit kept %v", got.corrections)
	}
	got, err = proofreadScan(t.Context(), reply.ask, numbered.ScanInstruction, pages, 0.9)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.corrections) != 1 {
		t.Errorf("a limit of 0.9 kept %v", got.corrections)
	}
}

func speech(count int) []numbered.Line {
	out := make([]numbered.Line, count)
	for item := range out {
		out[item] = numbered.Line{Number: item, Text: "word " + strconv.Itoa(item)}
	}
	return out
}

func TestProofreadSpeech_UsesSpeechInstruction(t *testing.T) {
	m := newReplyModel("")
	got, err := proofreadSpeech(t.Context(), m.ask, numbered.SpeechInstruction, speech(3), batchShape{size: 10, overlap: 0}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !got.isDone || got.next != 3 || got.uncorrected != 0 {
		t.Errorf("result %+v", got)
	}
	if len(m.prompts) != 1 || m.instructions[0] != numbered.SpeechInstruction {
		t.Fatalf("asked %d times under %q", len(m.prompts), m.instructions)
	}
	if !strings.HasPrefix(m.prompts[0], "The speech opens: word 0 word 1 word 2\n\n") {
		t.Errorf("prompt %q", m.prompts[0])
	}
}

func TestProofreadSpeech_ProcessesWindowsAndReturnsNext(t *testing.T) {
	m := newReplyModel("")
	got, err := proofreadSpeech(t.Context(), m.ask, numbered.SpeechInstruction, speech(20), batchShape{size: 4, overlap: 1}, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.isDone {
		t.Error("a window short of the end said it was done")
	}
	if got.next != 7 {
		t.Errorf("continue from %d, want the line shared with the end of the window, 7", got.next)
	}
	// The window's lines 0-7 are cut by 4 sharing 1: 0-3, 3-6, 6-7.
	if len(m.prompts) < 3 {
		t.Errorf("asked %d times", len(m.prompts))
	}

	got, err = proofreadSpeech(t.Context(), m.ask, numbered.SpeechInstruction, speech(20), batchShape{size: 4, overlap: 1}, got.next, 8)
	if err != nil || got.next != 14 {
		t.Errorf("second window: %+v, %v", got, err)
	}
	got, err = proofreadSpeech(t.Context(), m.ask, numbered.SpeechInstruction, speech(20), batchShape{size: 4, overlap: 1}, 14, 8)
	if err != nil || !got.isDone || got.next != 20 {
		t.Errorf("last window: %+v, %v", got, err)
	}
}

func TestSpeechPastTheEndIsCompleted(t *testing.T) {
	got, err := proofreadSpeech(t.Context(), newReplyModel("").ask, numbered.SpeechInstruction, speech(3), batchShape{size: 4, overlap: 1}, 9, 8)
	if err != nil || !got.isDone || got.next != 9 {
		t.Errorf("result %+v, %v", got, err)
	}
}

func TestProofreadSpeech_RetriesSplitBatchForSpanningRun(t *testing.T) {
	request := 0
	m := &model{script: func(_ string) string {
		request++
		if request == 1 {
			return "2-9|a run reaching past the first batch"
		}
		return ""
	}}
	got, err := proofreadSpeech(t.Context(), m.ask, numbered.SpeechInstruction, speech(8), batchShape{size: 4, overlap: 1}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Batches 0-3, 3-6, 6-7, then one window over the first cut.
	if len(m.prompts) != 4 || got.uncorrected != 0 || !got.isDone {
		t.Errorf("asked %d times, result %+v", len(m.prompts), got)
	}
}

func TestProofreadSpeech_RetriesWindowForSpanningRun(t *testing.T) {
	m := &model{script: func(prompt string) string {
		if strings.Contains(prompt, numbered.Opens+"7"+numbered.Closes) {
			return "6-9|a sentence reaching past the window"
		}
		return ""
	}}
	got, err := proofreadSpeech(t.Context(), m.ask, numbered.SpeechInstruction, speech(20), batchShape{size: 4, overlap: 1}, 0, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.next != 6 {
		t.Errorf("continue from %d, want the line the run begins on, 6", got.next)
	}
}

func TestProofreadSpeech_ReturnsCorrectionsInLineOrder(t *testing.T) {
	lines := []numbered.Line{{Number: 0, Text: "we should go. And then"}, {Number: 1, Text: "the next day he left."}, {Number: 2, Text: "the end"}}
	m := newReplyModel("0-1|We should go. And then the next day he left.\n2|The end")
	got, err := proofreadSpeech(t.Context(), m.ask, numbered.SpeechInstruction, lines, batchShape{size: 10, overlap: 0}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []numbered.Line{
		{Number: 0, Last: 1, Text: "We should go. And then the next day he left."},
		{Number: 2, Last: 2, Text: "The end"},
	}
	if !reflect.DeepEqual(got.corrections, want) {
		t.Errorf("corrections %+v, want %+v", got.corrections, want)
	}
}

func TestASpeechReplyTheGatesRefuseIsCountedUncorrected(t *testing.T) {
	got, err := proofreadSpeech(t.Context(), newReplyModel("999|invented").ask, numbered.SpeechInstruction, speech(3), batchShape{size: 10, overlap: 0}, 0, 0)
	if err != nil || got.uncorrected != 1 || len(got.corrections) != 0 {
		t.Errorf("result %+v, %v", got, err)
	}
}

func TestEveryScanRequestCarriesTheInstructionItIsGiven(t *testing.T) {
	m := newReplyModel("")
	given := numbered.BuildInstruction(numbered.ScanInstruction, "Hindi", "Keep names.") + " (given)"

	if _, err := proofreadScan(t.Context(), m.ask, given, twoPages(), 0); err != nil {
		t.Fatal(err)
	}

	if len(m.instructions) != 2 {
		t.Fatalf("asked %d times, want once a page", len(m.instructions))
	}
	for i, got := range m.instructions {
		if got != given {
			t.Errorf("request %d carried %q, want the one given", i, got)
		}
	}
}

func TestEveryRequestOfBothSpeechPassesCarriesTheInstructionItIsGiven(t *testing.T) {
	m := newReplyModel("")
	given := numbered.BuildInstruction(numbered.SpeechInstruction, "Hindi", "Keep names.") + " (given)"

	if _, err := proofreadSpeech(t.Context(), m.ask, given, speech(20), batchShape{size: 4, overlap: 1}, 0, 8); err != nil {
		t.Fatal(err)
	}

	if len(m.instructions) < 2 {
		t.Fatalf("asked %d times, want several", len(m.instructions))
	}
	for i, got := range m.instructions {
		if got != given {
			t.Errorf("request %d carried %q, want the one given", i, got)
		}
	}
}
