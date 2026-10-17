package main

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// model is a model whose reply is whatever its script makes of the prompt, and
// which remembers what it was asked.
type model struct {
	script       func(prompt string) string
	instructions []string
	prompts      []string
}

func (m *model) ask(_ context.Context, instruction, prompt string) (string, error) {
	m.instructions = append(m.instructions, instruction)
	m.prompts = append(m.prompts, prompt)
	return m.script(prompt), nil
}

// says is a model answering every prompt with the same reply.
func says(reply string) *model {
	return &model{script: func(string) string { return reply }}
}

func twoPages() []Batch {
	return []Batch{
		{Number: 3, Lines: []Line{{Number: 10, Text: "the cta sat"}, {Number: 11, Text: "on the mat"}}},
		{Number: 4, Lines: []Line{{Number: 12, Text: "the dgo ran"}}},
	}
}

func TestAScanIsAskedPageByPageAndCorrectionsComeBackInLineOrder(t *testing.T) {
	m := &model{script: func(prompt string) string {
		if strings.Contains(prompt, "dgo") {
			return "12|the dog ran"
		}
		return "10|the cat sat"
	}}

	got, err := proofreadScan(t.Context(), m.ask, twoPages(), textOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Line{{Number: 10, Text: "the cat sat"}, {Number: 12, Text: "the dog ran"}}
	if !reflect.DeepEqual(got.corrections, want) {
		t.Errorf("corrections %v, want %v", got.corrections, want)
	}
	if got.uncorrected != 0 || !got.isDone {
		t.Errorf("result %+v", got)
	}
	if len(m.prompts) != 2 || m.instructions[0] != scanInstruction {
		t.Errorf("asked %d times under %q", len(m.prompts), m.instructions)
	}
}

func TestAScanPageWhoseReplyIsRefusedIsCountedUncorrected(t *testing.T) {
	got, err := proofreadScan(t.Context(), says("999|invented").ask, twoPages(), textOptions{})
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
	if _, err := proofreadScan(t.Context(), ask, twoPages(), textOptions{}); !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestTheEditLimitOfARequestIsHeldAndStrictWidensIt(t *testing.T) {
	pages := []Batch{{Number: 1, Lines: []Line{{Number: 1, Text: "abcdefghij"}}}}
	reply := says("1|abcdeXYZ")

	got, _ := proofreadScan(t.Context(), reply.ask, pages, textOptions{})
	if len(got.corrections) != 0 {
		t.Errorf("default limit kept %v", got.corrections)
	}
	got, _ = proofreadScan(t.Context(), reply.ask, pages, textOptions{maxDistance: 0.9})
	if len(got.corrections) != 1 {
		t.Errorf("a limit of 0.9 kept %v", got.corrections)
	}
	got, _ = proofreadScan(t.Context(), reply.ask, pages, textOptions{strict: true})
	if len(got.corrections) != 1 {
		t.Errorf("strict kept %v", got.corrections)
	}
	if !strings.HasPrefix(reply.instructions[len(reply.instructions)-1], scanInstruction) ||
		reply.instructions[len(reply.instructions)-1] == scanInstruction {
		t.Error("strict adds its rule to the instruction")
	}
}

func TestAScanKeepsNoMoreCorrectionsThanAreAllowedToAPage(t *testing.T) {
	pages := []Batch{{Number: 1, Lines: []Line{
		{Number: 1, Text: "the cta sat on the mat"},
		{Number: 2, Text: "the dgo sat on the mat"},
	}}}
	reply := says("1|the cat sat on the mat\n2|the dog sat on the mat")
	got, _ := proofreadScan(t.Context(), reply.ask, pages, textOptions{maxSuggestions: 1})
	if len(got.corrections) != 1 || got.corrections[0].Number != 1 {
		t.Errorf("corrections %v", got.corrections)
	}
}

// speech is a transcript of count lines, line n saying "word n".
func speech(count int) []Line {
	out := make([]Line, count)
	for at := range out {
		out[at] = Line{Number: at, Text: "word " + strconv.Itoa(at)}
	}
	return out
}

func TestSpeechIsAskedUnderTheSpeechInstructionWithWhatTheRecordingHolds(t *testing.T) {
	m := says("")
	got, err := proofreadSpeech(t.Context(), m.ask, speech(3), 10, 0, 0, 0, textOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !got.isDone || got.next != 3 || got.uncorrected != 0 {
		t.Errorf("result %+v", got)
	}
	if len(m.prompts) != 1 || m.instructions[0] != speechInstruction {
		t.Fatalf("asked %d times under %q", len(m.prompts), m.instructions)
	}
	if !strings.HasPrefix(m.prompts[0], "The speech opens: word 0 word 1 word 2\n\n") {
		t.Errorf("prompt %q", m.prompts[0])
	}
}

func TestSpeechIsAnsweredAWindowAtATimeAndSaysWhereToContinue(t *testing.T) {
	m := says("")
	got, err := proofreadSpeech(t.Context(), m.ask, speech(20), 4, 1, 0, 8, textOptions{})
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

	got, err = proofreadSpeech(t.Context(), m.ask, speech(20), 4, 1, got.next, 8, textOptions{})
	if err != nil || got.next != 14 {
		t.Errorf("second window: %+v, %v", got, err)
	}
	got, err = proofreadSpeech(t.Context(), m.ask, speech(20), 4, 1, 14, 8, textOptions{})
	if err != nil || !got.isDone || got.next != 20 {
		t.Errorf("last window: %+v, %v", got, err)
	}
}

func TestSpeechPastTheEndIsDone(t *testing.T) {
	got, err := proofreadSpeech(t.Context(), says("").ask, speech(3), 4, 1, 9, 8, textOptions{})
	if err != nil || !got.isDone || got.next != 9 {
		t.Errorf("result %+v, %v", got, err)
	}
}

func TestACutBatchIsAskedWhereARunReachedPastABatch(t *testing.T) {
	asked := 0
	m := &model{script: func(prompt string) string {
		asked++
		if asked == 1 {
			return "2-9|a run reaching past the first batch"
		}
		return ""
	}}
	got, err := proofreadSpeech(t.Context(), m.ask, speech(8), 4, 1, 0, 0, textOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// Batches 0-3, 3-6, 6-7, then one window over the first cut.
	if len(m.prompts) != 4 || got.uncorrected != 0 || !got.isDone {
		t.Errorf("asked %d times, result %+v", len(m.prompts), got)
	}
}

func TestARunReachingPastTheWindowIsAskedAgainFromWhereItBegins(t *testing.T) {
	m := &model{script: func(prompt string) string {
		if strings.Contains(prompt, Opens+"7"+Closes) {
			return "6-9|a sentence reaching past the window"
		}
		return ""
	}}
	got, err := proofreadSpeech(t.Context(), m.ask, speech(20), 4, 1, 0, 8, textOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.next != 6 {
		t.Errorf("continue from %d, want the line the run begins on, 6", got.next)
	}
}

func TestSpeechCorrectionsComeBackInLineOrderAndAJoinedRunKeepsItsLast(t *testing.T) {
	lines := []Line{{Number: 0, Text: "we should go. And then"}, {Number: 1, Text: "the next day he left."}, {Number: 2, Text: "the end"}}
	m := says("0-1|We should go. And then the next day he left.\n2|The end")
	got, err := proofreadSpeech(t.Context(), m.ask, lines, 10, 0, 0, 0, textOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Line{
		{Number: 0, Last: 1, Text: "We should go. And then the next day he left."},
		{Number: 2, Text: "The end"},
	}
	if !reflect.DeepEqual(got.corrections, want) {
		t.Errorf("corrections %+v, want %+v", got.corrections, want)
	}
}

func TestASpeechReplyTheGatesRefuseIsCountedUncorrected(t *testing.T) {
	got, err := proofreadSpeech(t.Context(), says("999|invented").ask, speech(3), 10, 0, 0, 0, textOptions{})
	if err != nil || got.uncorrected != 1 || len(got.corrections) != 0 {
		t.Errorf("result %+v, %v", got, err)
	}
}
