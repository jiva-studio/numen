package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/ai"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// host is a completer that answers every question through its script and keeps
// what it was asked.
type host struct {
	script func(prompt string) string
	asked  []*numenv1.CompletionQuestion
}

func (h *host) complete(_ context.Context, question *numenv1.CompletionQuestion) (string, error) {
	h.asked = append(h.asked, question)
	return h.script(question.GetMessages()[1].GetText()), nil
}

func newServer(script func(prompt string) string) (*proofreadingServer, *host) {
	h := &host{script: script}
	return newProofreadingServer(h.complete), h
}

func scanRequest(distance float64, groups ...*numenv1.ProofreadGroup) *numenv1.ProofreadTextRequest {
	return &numenv1.ProofreadTextRequest{
		Kind:   numenv1.ProofreadKind_PROOFREAD_KIND_SCAN,
		Groups: groups,
		Options: &numenv1.ProofreadTextRequest_Scan{
			Scan: &numenv1.ProofreadScanOptions{MaxEditDistance: distance},
		},
	}
}

func speechRequest(options *numenv1.ProofreadSpeechOptions, lines ...*numenv1.ProofreadLine) *numenv1.ProofreadTextRequest {
	return &numenv1.ProofreadTextRequest{
		Kind:    numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH,
		Groups:  []*numenv1.ProofreadGroup{{Lines: lines}},
		Options: &numenv1.ProofreadTextRequest_Speech{Speech: options},
	}
}

func page(number int32, lines ...*numenv1.ProofreadLine) *numenv1.ProofreadGroup {
	return &numenv1.ProofreadGroup{Number: number, Lines: lines}
}

func line(number int32, text string) *numenv1.ProofreadLine {
	return &numenv1.ProofreadLine{Number: number, Text: text}
}

func TestGetProofreaderInfoNamesTheProofreader(t *testing.T) {
	proof, _ := newServer(nil)
	info, err := proof.GetProofreaderInfo(t.Context(), connect.NewRequest(&numenv1.GetProofreaderInfoRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Msg.GetName(), "llm-proofreader"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
}

func TestTheModelIsAskedByPurposeAloneAtZeroTemperature(t *testing.T) {
	proof, h := newServer(func(string) string { return "" })

	if _, err := proof.ask(t.Context(), "be careful", "the prompt"); err != nil {
		t.Fatal(err)
	}

	if len(h.asked) != 1 {
		t.Fatalf("asked %d times", len(h.asked))
	}
	q := h.asked[0]
	if q.GetPurpose() != "proofreading" {
		t.Errorf("purpose %q", q.GetPurpose())
	}
	if len(q.GetMessages()) != 2 ||
		q.GetMessages()[0].GetRole() != "system" || q.GetMessages()[0].GetText() != "be careful" ||
		q.GetMessages()[1].GetRole() != "user" || q.GetMessages()[1].GetText() != "the prompt" {
		t.Errorf("messages %v", q.GetMessages())
	}
	if q.GetHints() == nil || q.GetHints().Temperature == nil || q.GetHints().GetTemperature() != 0 {
		t.Errorf("hints %v", q.GetHints())
	}
}

func TestProofreadTextAnswersAScan(t *testing.T) {
	proof, h := newServer(func(prompt string) string {
		if strings.Contains(prompt, "dgo") {
			return "12|the dog ran"
		}
		return "10|the cat sat"
	})

	res, err := proof.ProofreadText(t.Context(), connect.NewRequest(scanRequest(0,
		page(3, line(10, "the cta sat"), line(11, "on the mat")),
		page(4, line(12, "the dgo ran")),
	)))
	if err != nil {
		t.Fatal(err)
	}

	got := res.Msg.GetCorrections()
	if len(got) != 2 || got[0].GetNumber() != 10 || got[0].GetText() != "the cat sat" ||
		got[1].GetNumber() != 12 || got[1].GetText() != "the dog ran" {
		t.Errorf("corrections %v", got)
	}
	if !res.Msg.GetIsDone() || res.Msg.GetUncorrected() != 0 {
		t.Errorf("response %v", res.Msg)
	}
	if len(h.asked) != 2 || h.asked[0].GetMessages()[0].GetText() != scanInstruction {
		t.Errorf("asked %d times", len(h.asked))
	}
}

func TestProofreadTextHoldsAScanToTheDistanceItIsGiven(t *testing.T) {
	proof, _ := newServer(func(string) string { return "1|abcdeXYZ" })
	request := func(distance float64) int {
		res, err := proof.ProofreadText(t.Context(), connect.NewRequest(
			scanRequest(distance, page(1, line(1, "abcdefghij")))))
		if err != nil {
			t.Fatal(err)
		}
		return len(res.Msg.GetCorrections())
	}

	if kept := request(0); kept != 0 {
		t.Errorf("the proofreader's own limit kept %d", kept)
	}
	if kept := request(0.9); kept != 1 {
		t.Errorf("a limit of 0.9 kept %d", kept)
	}
}

func TestProofreadTextAnswersSpeechAWindowAtATimeAndKeepsAJoinedRun(t *testing.T) {
	proof, _ := newServer(func(prompt string) string {
		if strings.Contains(prompt, Opens+"0"+Closes) {
			return "0-1|We should go. And then the next day he left."
		}
		return ""
	})
	lines := []*numenv1.ProofreadLine{
		line(0, "we should go. And then"), line(1, "the next day he left."), line(2, "the end"),
	}

	res, err := proof.ProofreadText(t.Context(), connect.NewRequest(
		speechRequest(&numenv1.ProofreadSpeechOptions{Size: 10}, lines...)))
	if err != nil {
		t.Fatal(err)
	}

	got := res.Msg.GetCorrections()
	if len(got) != 1 || got[0].GetNumber() != 0 || got[0].GetLast() != 1 {
		t.Errorf("corrections %v", got)
	}
	if !res.Msg.GetIsDone() || res.Msg.GetNextLine() != 3 {
		t.Errorf("response %v", res.Msg)
	}

	res, err = proof.ProofreadText(t.Context(), connect.NewRequest(
		speechRequest(&numenv1.ProofreadSpeechOptions{Size: 2, Overlap: 0, LineLimit: 2}, lines...)))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetIsDone() || res.Msg.GetNextLine() != 2 {
		t.Errorf("a window short of the end: %v", res.Msg)
	}
}

func TestProofreadTextRefusesARequestItCannotAnswer(t *testing.T) {
	proof, h := newServer(func(string) string { return "" })
	cases := map[string]*numenv1.ProofreadTextRequest{
		"no kind": {Groups: []*numenv1.ProofreadGroup{page(1, line(1, "x"))}},
		"scan without scan options": {
			Kind:    numenv1.ProofreadKind_PROOFREAD_KIND_SCAN,
			Options: &numenv1.ProofreadTextRequest_Speech{Speech: &numenv1.ProofreadSpeechOptions{}},
		},
		"scan without options": {Kind: numenv1.ProofreadKind_PROOFREAD_KIND_SCAN},
		"speech with scan options": {
			Kind:    numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH,
			Options: &numenv1.ProofreadTextRequest_Scan{Scan: &numenv1.ProofreadScanOptions{}},
		},
		"speech without options": {Kind: numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH},
	}
	for name, request := range cases {
		_, err := proof.ProofreadText(t.Context(), connect.NewRequest(request))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: got %v, want invalid argument", name, err)
		}
	}
	if len(h.asked) != 0 {
		t.Errorf("a refused request asked the model %d times", len(h.asked))
	}
}

func TestProofreadTextHandlesEveryKindTheSchemaNames(t *testing.T) {
	proof, _ := newServer(func(string) string { return "" })
	for value, name := range numenv1.ProofreadKind_name {
		kind := numenv1.ProofreadKind(value)
		if kind == numenv1.ProofreadKind_PROOFREAD_KIND_UNSPECIFIED {
			continue
		}
		request := &numenv1.ProofreadTextRequest{
			Kind:   kind,
			Groups: []*numenv1.ProofreadGroup{page(1, line(1, "words"))},
		}
		switch kind {
		case numenv1.ProofreadKind_PROOFREAD_KIND_SCAN:
			request.Options = &numenv1.ProofreadTextRequest_Scan{Scan: &numenv1.ProofreadScanOptions{}}
		case numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH:
			request.Options = &numenv1.ProofreadTextRequest_Speech{Speech: &numenv1.ProofreadSpeechOptions{Size: 4}}
		case numenv1.ProofreadKind_PROOFREAD_KIND_UNSPECIFIED:
		}
		if _, err := proof.ProofreadText(t.Context(), connect.NewRequest(request)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestProofreadAnswersEachBatchUnderTheInstructionItIsGiven(t *testing.T) {
	proof, h := newServer(func(prompt string) string { return "corrected " + prompt })

	res, err := proof.Proofread(t.Context(), connect.NewRequest(&numenv1.ProofreadRequest{
		Instruction: "be exact",
		Batches:     []*numenv1.ProofreadBatch{{Number: 4, Prompt: "one"}, {Number: 9, Prompt: "two"}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if got := res.Msg.GetReplies(); len(got) != 2 || got[4] != "corrected one" || got[9] != "corrected two" {
		t.Errorf("replies %v", got)
	}
	if h.asked[0].GetMessages()[0].GetText() != "be exact" {
		t.Errorf("instruction %q", h.asked[0].GetMessages()[0].GetText())
	}
}

func TestAModelThatCannotBeReachedIsUnavailable(t *testing.T) {
	cases := map[string]error{
		"no route":    ai.ErrNoRoute,
		"unavailable": ai.ErrUnavailable,
		"no host":     ai.ErrNoHost,
		"wrapped":     fmt.Errorf("batch 1: %w", ai.ErrNoRoute),
	}
	for name, failure := range cases {
		proof := newProofreadingServer(func(context.Context, *numenv1.CompletionQuestion) (string, error) {
			return "", failure
		})
		_, err := proof.ProofreadText(t.Context(), connect.NewRequest(scanRequest(0, page(1, line(1, "words")))))
		if connect.CodeOf(err) != connect.CodeUnavailable || !errors.Is(err, failure) {
			t.Errorf("%s: got %v, want unavailable", name, err)
		}
	}
}

func TestAFailureOfTheModelThatIsNotAbsenceIsInternal(t *testing.T) {
	boom := errors.New("boom")
	proof := newProofreadingServer(func(context.Context, *numenv1.CompletionQuestion) (string, error) {
		return "", boom
	})

	_, err := proof.ProofreadText(t.Context(), connect.NewRequest(scanRequest(0, page(1, line(1, "words")))))

	if connect.CodeOf(err) != connect.CodeInternal || !errors.Is(err, boom) {
		t.Errorf("got %v", err)
	}
}

func TestACancelledCallIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	proof := newProofreadingServer(func(context.Context, *numenv1.CompletionQuestion) (string, error) {
		cancel()
		return "", errors.New("interrupted")
	})

	_, err := proof.ProofreadText(ctx, connect.NewRequest(scanRequest(0, page(1, line(1, "words")))))

	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v", err)
	}
}

func TestWithoutAHostTheProofreadingFailsAndInventsNothing(t *testing.T) {
	t.Setenv(ai.HostPortEnv, "")
	proof := newProofreadingServer(ai.Complete)

	res, err := proof.ProofreadText(t.Context(), connect.NewRequest(scanRequest(0, page(1, line(1, "words")))))

	if connect.CodeOf(err) != connect.CodeUnavailable || res != nil {
		t.Errorf("got %v, %v", res, err)
	}
}
