package proofread_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// model is a model whose reply is whatever its script makes of the prompt, and
// which remembers what it was asked.
type model struct {
	script func(prompt string) string

	mu           sync.Mutex
	instructions []string
	prompts      []string
}

func (m *model) ask(_ context.Context, instruction, prompt string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instructions = append(m.instructions, instruction)
	m.prompts = append(m.prompts, prompt)
	return m.script(prompt), nil
}

func says(reply string) *model {
	return &model{script: func(string) string { return reply }}
}

func twoPages() []domain.ProofreadGroup {
	return []domain.ProofreadGroup{
		{Number: 3, Lines: []domain.ProofreadLine{{Number: 10, Text: "the cta sat"}, {Number: 11, Text: "on the mat"}}},
		{Number: 4, Lines: []domain.ProofreadLine{{Number: 12, Text: "the dgo ran"}}},
	}
}

func TestAScanIsAskedPageByPageAndCorrectionsComeBackInLineOrder(t *testing.T) {
	m := &model{script: func(prompt string) string {
		if strings.Contains(prompt, "dgo") {
			return "12|the dog ran"
		}
		return "10|the cat sat"
	}}

	got, err := proofread.Scan(t.Context(), m.ask, twoPages(), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.ProofreadCorrection{{Number: 10, Last: 0, Text: "the cat sat"}, {Number: 12, Text: "the dog ran"}}
	if !reflect.DeepEqual(got.Corrections, want) {
		t.Errorf("corrections %v, want %v", got.Corrections, want)
	}
	if got.Uncorrected != 0 || !got.IsDone {
		t.Errorf("answer %+v", got)
	}
	if len(m.prompts) != 2 || m.instructions[0] != proofread.ScanInstruction {
		t.Errorf("asked %d times under %q", len(m.prompts), m.instructions)
	}
}

func TestAScanPageWhoseReplyIsRefusedIsCountedUncorrected(t *testing.T) {
	got, err := proofread.Scan(t.Context(), says("999|invented").ask, twoPages(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Uncorrected != 2 || len(got.Corrections) != 0 {
		t.Errorf("answer %+v", got)
	}
}

func TestAScanFailureEndsTheRun(t *testing.T) {
	boom := errors.New("boom")
	ask := func(context.Context, string, string) (string, error) { return "", boom }
	if _, err := proofread.Scan(t.Context(), ask, twoPages(), 0); !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestTheEditLimitOfARequestIsHeld(t *testing.T) {
	pages := []domain.ProofreadGroup{{Number: 1, Lines: []domain.ProofreadLine{{Number: 1, Text: "abcdefghij"}}}}
	reply := says("1|abcdeXYZ")

	if got, _ := proofread.Scan(t.Context(), reply.ask, pages, 0); len(got.Corrections) != 0 {
		t.Errorf("the default limit kept %v", got.Corrections)
	}
	if got, _ := proofread.Scan(t.Context(), reply.ask, pages, 0.9); len(got.Corrections) != 1 {
		t.Errorf("a limit of 0.9 kept %v", got.Corrections)
	}
}

// speech is a transcript of count lines, line n saying "word n".
func speech(count int) []domain.ProofreadGroup {
	lines := make([]domain.ProofreadLine, count)
	for at := range lines {
		lines[at] = domain.ProofreadLine{Number: at, Text: "word " + string(rune('a'+at))}
	}
	return []domain.ProofreadGroup{{Lines: lines}}
}

func request(count, size, overlap, from, limit int) domain.ProofreadRequest {
	return domain.ProofreadRequest{
		Kind: domain.ProofreadSpeech, Groups: speech(count),
		Size: size, Overlap: overlap, FromLine: from, LineLimit: limit,
	}
}

func TestSpeechIsAskedUnderTheSpeechInstructionWithWhatTheRecordingHolds(t *testing.T) {
	m := says("")
	got, err := proofread.Speech(t.Context(), m.ask, request(3, 10, 0, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsDone || got.NextLine != 3 || got.Uncorrected != 0 {
		t.Errorf("answer %+v", got)
	}
	if len(m.prompts) != 1 || m.instructions[0] != proofread.SpeechInstruction {
		t.Fatalf("asked %d times under %q", len(m.prompts), m.instructions)
	}
	if !strings.HasPrefix(m.prompts[0], "The speech opens: word a word b word c\n") {
		t.Errorf("prompt %q", m.prompts[0])
	}
}

func TestSpeechIsAnsweredAWindowAtATimeAndSaysWhereToContinue(t *testing.T) {
	m := says("")
	// 20 lines by 4 sharing 1 are batches opening on 0, 3, 6, 9, 12, 15 and 18.
	got, err := proofread.Speech(t.Context(), m.ask, request(20, 4, 1, 0, 8))
	if err != nil {
		t.Fatal(err)
	}
	if got.IsDone || got.NextLine != 6 || len(m.prompts) != 2 {
		t.Errorf("answer %+v after %d requests, want the next batch to open on line 6", got, len(m.prompts))
	}
	got, err = proofread.Speech(t.Context(), m.ask, request(20, 4, 1, got.NextLine, 8))
	if err != nil || got.NextLine != 12 {
		t.Errorf("second window: %+v, %v", got, err)
	}
	got, err = proofread.Speech(t.Context(), m.ask, request(20, 4, 1, got.NextLine, 8))
	if err != nil || got.NextLine != 18 || got.IsDone {
		t.Errorf("third window: %+v, %v", got, err)
	}
	got, err = proofread.Speech(t.Context(), m.ask, request(20, 4, 1, got.NextLine, 8))
	if err != nil || !got.IsDone || got.NextLine != 20 {
		t.Errorf("last window: %+v, %v", got, err)
	}
}

func TestSpeechPastTheEndIsDone(t *testing.T) {
	got, err := proofread.Speech(t.Context(), says("").ask, request(3, 4, 1, 9, 8))
	if err != nil || !got.IsDone || got.NextLine != 9 {
		t.Errorf("answer %+v, %v", got, err)
	}
}

func TestAWindowBeginningInTheMiddleOfABatchAsksTheBatchHoldingIt(t *testing.T) {
	m := says("")
	got, err := proofread.Speech(t.Context(), m.ask, request(8, 4, 1, 4, 0))
	if err != nil {
		t.Fatal(err)
	}
	// Batches open on 0 and 3; line 4 is held by the second, and by the third on 6.
	if !got.IsDone || len(m.prompts) != 2 {
		t.Errorf("answer %+v after %d requests", got, len(m.prompts))
	}
}

func TestACutBatchIsAskedWhereARunReachedPastABatch(t *testing.T) {
	m := &model{script: func(prompt string) string {
		if strings.Contains(prompt, proofread.Opens+"0"+proofread.Closes) {
			return "2-9|a run reaching past the first batch"
		}
		return ""
	}}
	// Batches open on 0, 3 and 6, and a window over the first cut makes four requests.
	got, err := proofread.Speech(t.Context(), m.ask, request(8, 4, 1, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.prompts) != 4 || got.Uncorrected != 0 || !got.IsDone {
		t.Errorf("asked %d times, answer %+v", len(m.prompts), got)
	}
}

func TestSpeechCorrectionsComeBackInLineOrderAndAJoinedRunKeepsItsLast(t *testing.T) {
	groups := []domain.ProofreadGroup{{Lines: []domain.ProofreadLine{
		{Number: 0, Text: "we should go. And then"},
		{Number: 1, Text: "the next day he left."},
		{Number: 2, Text: "the end"},
	}}}
	m := says("0-1|We should go. And then the next day he left.\n2|The end")
	got, err := proofread.Speech(t.Context(), m.ask, domain.ProofreadRequest{Groups: groups, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.ProofreadCorrection{
		{Number: 0, Last: 1, Text: "We should go. And then the next day he left."},
		{Number: 2, Last: 2, Text: "The end"},
	}
	if !reflect.DeepEqual(got.Corrections, want) {
		t.Errorf("corrections %+v, want %+v", got.Corrections, want)
	}
}
