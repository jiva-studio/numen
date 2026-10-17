package main

import (
	"strings"
	"testing"
)

const scanPrompt = "Context line.\n\n" + Opens + "10" + Closes + "the Sodërby gardin hcdge. " + Opens + "11" + Closes + "blessings." + Opens + "12" + Closes + "The hedge will need cutting"

func TestReadPromptReturnsEveryNumberedLine(t *testing.T) {
	lines := readPrompt(scanPrompt)
	if len(lines) != 3 || lines[0].Number != 10 || lines[2].Number != 12 {
		t.Fatalf("lines = %v", lines)
	}
	if lines[0].Text != "the Sodërby gardin hcdge." {
		t.Errorf("text = %q", lines[0].Text)
	}
}

func TestReadPromptWithoutMarksHasNoLines(t *testing.T) {
	if lines := readPrompt("plain text"); len(lines) != 0 {
		t.Errorf("lines = %v", lines)
	}
}

func TestReviewReplyKeepsAcceptedCorrections(t *testing.T) {
	got := reviewReply(scanPrompt, "10|the Södërby garden hedge.", reviewOptions{})
	if got != "10|the Södërby garden hedge." {
		t.Errorf("reply = %q", got)
	}
}

func TestReviewReplyRefusedByTheGatesIsEmpty(t *testing.T) {
	if got := reviewReply(scanPrompt, "999|invented", reviewOptions{}); got != "" {
		t.Errorf("reply = %q, want empty", got)
	}
}

func TestReviewReplyDropsACorrectionBeyondTheEditLimit(t *testing.T) {
	far := "11|completely different words entirely here"
	if got := reviewReply(scanPrompt, far, reviewOptions{}); got != "" {
		t.Errorf("reply = %q, want empty", got)
	}
}

func TestStrictModeWidensTheEditLimit(t *testing.T) {
	was := "abcdefghij"
	prompt := Opens + "1" + Closes + was
	reply := "1|abcdeXYZ"
	if got := reviewReply(prompt, reply, reviewOptions{}); got != "" {
		t.Fatalf("lenient reply = %q, want empty", got)
	}
	if got := reviewReply(prompt, reply, reviewOptions{strict: true}); got != reply {
		t.Errorf("strict reply = %q, want %q", got, reply)
	}
}

func TestMaxSuggestionsCapsTheCorrections(t *testing.T) {
	prompt := Opens + "1" + Closes + "the cta sat on the mat " + Opens + "2" + Closes + "the dgo sat on the mat " + Opens + "3" + Closes + "the brid sat on the mat"
	reply := "1|the cat sat on the mat\n2|the dog sat on the mat\n3|the bird sat on the mat"
	got := reviewReply(prompt, reply, reviewOptions{maxSuggestions: 2})
	if got != "1|the cat sat on the mat\n2|the dog sat on the mat" {
		t.Errorf("reply = %q", got)
	}
}

func TestZeroMaxSuggestionsSetsNoCap(t *testing.T) {
	prompt := Opens + "1" + Closes + "the cta sat on the mat " + Opens + "2" + Closes + "the dgo sat on the mat"
	reply := "1|the cat sat on the mat\n2|the dog sat on the mat"
	if got := reviewReply(prompt, reply, reviewOptions{}); got != reply {
		t.Errorf("reply = %q", got)
	}
}

func TestReviewReplyKeepsARunReachingPastTheBatch(t *testing.T) {
	prompt := Opens + "4" + Closes + "Krishna is" + Opens + "5" + Closes + "connected with Raj Dila."
	reply := "4-5|Krishna is connected with Radha Dila.\n5-9|runs on past the end"
	got := reviewReply(prompt, reply, reviewOptions{})
	if !strings.Contains(got, "5-9|runs on past the end") || !strings.Contains(got, "4-5|") {
		t.Errorf("reply = %q", got)
	}
}

func TestBuildInstructionAddsStrictRuleOnlyInStrictMode(t *testing.T) {
	if got := buildInstruction("base", false); got != "base" {
		t.Errorf("lenient = %q", got)
	}
	if got := buildInstruction("base", true); !strings.HasPrefix(got, "base") || len(got) <= len("base") {
		t.Errorf("strict = %q", got)
	}
}
