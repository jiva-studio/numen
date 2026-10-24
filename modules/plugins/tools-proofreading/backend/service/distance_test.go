package service

import "testing"

// A line of speech, and the words a proofreader answers with.
const (
	asHeard = "Shur Maheshvasa M Vima Janahana Sam Yud Yu Yudhano Yuratas Drupadasha Duru Maharataha."
	asSaid  = "Shura Maheshvasa Vikranta Uttamauja Yudhamanyu Yuyudhana Drupada Maharatha."
)

func oneLine(text string) Batch {
	return Batch{Number: 1, Lines: []Line{{Number: 7, Text: text}}}
}

// right.
func TestACorrectionHeldToNoDistanceExistsHoweverFar(t *testing.T) {
	if apart := EditDistance(asHeard, asSaid); apart <= MaxEditDistance {
		t.Fatalf("the two stand %.2f apart, which the recognition's own limit allows", apart)
	}

	placed, _, ok := GetFixedLines(oneLine(asHeard), "7 | "+asSaid, 0)
	if !ok {
		t.Fatal("the batch failed")
	}
	if len(placed) != 1 || placed[0].Text != asSaid {
		t.Errorf("the line was put right to %v", placed)
	}
}

// The recognition's own limit still holds where it is given.
func TestACorrectionBeyondTheLimitGivenIsDropped(t *testing.T) {
	placed, _, ok := GetFixedLines(oneLine(asHeard), "7 | "+asSaid, MaxEditDistance)
	if !ok {
		t.Fatal("the batch failed")
	}
	if len(placed) != 0 {
		t.Errorf("the line was put right to %v", placed)
	}
}

// An answer that empties a line which said something refuses the batch.
func TestAnAnswerThatEmptiesALineRefusesTheBatch(t *testing.T) {
	if _, _, ok := GetFixedLines(oneLine(asHeard), "7 | ", 0); ok {
		t.Error("the batch was taken")
	}
}
