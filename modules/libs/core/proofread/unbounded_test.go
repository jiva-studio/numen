package proofread_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// A line of speech, and the words a proofreader answers with.
const (
	asHeard = "Shur Maheshvasa M Vima Janahana Sam Yud Yu Yudhano Yuratas Drupadasha Duru Maharataha."
	asSaid  = "Shura Maheshvasa Vikranta Uttamauja Yudhamanyu Yuyudhana Drupada Maharatha."
)

func oneLine(text string) proofread.Batch {
	return proofread.Batch{Number: 1, Lines: []proofread.Line{{Number: 7, Text: text}}}
}

// Held to no distance, a correction stands however far from the line it puts
// right.
func TestACorrectionHeldToNoDistanceStandsHoweverFar(t *testing.T) {
	if apart := proofread.EditDistance(asHeard, asSaid); apart <= proofread.MaxEditDistance {
		t.Fatalf("the two stand %.2f apart, which the reading's own limit allows", apart)
	}

	put, _, ok := proofread.Fixed(oneLine(asHeard), "7 | "+asSaid, 0)
	if !ok {
		t.Fatal("the batch was refused")
	}
	if len(put) != 1 || put[0].Text != asSaid {
		t.Errorf("the line was put right to %v", put)
	}
}

// The reading's own limit still holds where it is given.
func TestACorrectionBeyondTheLimitGivenIsDropped(t *testing.T) {
	put, _, ok := proofread.Fixed(oneLine(asHeard), "7 | "+asSaid, proofread.MaxEditDistance)
	if !ok {
		t.Fatal("the batch was refused")
	}
	if len(put) != 0 {
		t.Errorf("the line was put right to %v", put)
	}
}

// An answer that empties a line which said something refuses the batch.
func TestAnAnswerThatEmptiesALineRefusesTheBatch(t *testing.T) {
	if _, _, ok := proofread.Fixed(oneLine(asHeard), "7 | ", 0); ok {
		t.Error("the batch was taken")
	}
}
