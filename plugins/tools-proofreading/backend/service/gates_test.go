package service

import "testing"

// batch is a page as it was read, misreadings and all.
func batch() Batch {
	return Batch{Number: 7, Lines: []Line{
		{Number: 10, Text: "the Sodërby gardin hcdge. "},
		{Number: 11, Text: "Sodërby Tradgard's Handbok "},
		{Number: 12, Text: "TRADGARD HANDBOK "},
		{Number: 13, Text: "The hedge will very soon need cutting back "},
		{Number: 20, Text: "blessings."},
	}}
}

func TestGetFixedLines_RejectsReplyContainingMarker(t *testing.T) {
	reply := "10|the " + Opens + "12" + Closes + "Södërby garden hedge."

	fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance)
	if replyVal {
		t.Errorf("a reply carrying %s was taken as an answer", Opens)
	}
	if fixed != nil {
		t.Errorf("a rejected batch gave back %v", fixed)
	}
}

func TestGetFixedLines_RejectsUnknownLineNumber(t *testing.T) {
	reply := "10|the Södërby garden hedge.\n999|a line this batch never carried"

	fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance)
	if replyVal {
		t.Errorf("a reply naming line 999 was taken as an answer")
	}
	if fixed != nil {
		t.Errorf("a rejected batch gave back %v", fixed)
	}
}

func TestGetFixedLines_RejectsLineWithoutNumberAndBar(t *testing.T) {
	reply := "Here are the lines I would put right:\n10|the Södërby garden hedge."

	if fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance); replyVal {
		t.Errorf("a reply that talks was taken as an answer, giving %v", fixed)
	}
}

func TestAFencedReplyIsUnwrapped(t *testing.T) {
	reply := "```text\n10|the Södërby garden hedge.\n20|blessings!\n```"

	fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance)
	if !replyVal {
		t.Fatalf("a fenced reply failed")
	}
	if len(fixed) != 2 {
		t.Fatalf("%d lines out of a fenced reply, want two: %v", len(fixed), fixed)
	}
	if fixed[0].Number != 10 || fixed[0].Text != "the Södërby garden hedge." {
		t.Errorf("line %d says %q", fixed[0].Number, fixed[0].Text)
	}
}

func TestTheLettersOfACorrectionMayOnlyMoveSoFar(t *testing.T) {
	for _, one := range []struct {
		was, replacement string
		least, most      float64
		isKept           bool
	}{
		// A misread word, and diacritics the page prints.
		{"the Sodërby gardin hcdge.", "the Södërby garden hedge.", 0.05, 0.15, true},
		// Diacritics alone: folded, the two say the same letters.
		{"Sodërby Tradgard's Handbok", "Södërby Trädgård's Handbök", 0, 0, true},
		// A heading answered with the paragraph under it.
		{"TRADGARD HANDBOK", "TRÄDGÅRD HANDBÖK At the request of the Sodërby committee", 0.60, 0.75, false},
		// A line answered with one word of it.
		{"The hedge will very soon need cutting back", "hedge", 0.80, 0.95, false},
	} {
		apart := EditDistance(one.was, one.replacement)
		if apart < one.least || apart > one.most {
			t.Errorf("%q -> %q stands %.4f apart, want between %v and %v", one.was, one.replacement, apart, one.least, one.most)
		}
		if isAccepted := apart <= MaxEditDistance; isAccepted != one.isKept {
			t.Errorf("%q -> %q stands %.4f apart, kept %v, want kept %v", one.was, one.replacement, apart, isAccepted, one.isKept)
		}
	}
}

func TestGetFixedLines_ExcessiveEditDistanceDropped(t *testing.T) {
	reply := "10|the Södërby garden hedge.\n" +
		"11|Södërby Trädgård's Handbök\n" +
		"12|TRÄDGÅRD HANDBÖK At the request of the Sodërby committee\n" +
		"13|hedge"

	fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance)
	if !replyVal {
		t.Fatalf("one correction that moved too far failed the whole batch")
	}
	var numbers []int
	for _, line := range fixed {
		numbers = append(numbers, line.Number)
	}
	if len(numbers) != 2 || numbers[0] != 10 || numbers[1] != 11 {
		t.Errorf("lines %v put right, want the two whose letters stayed put", numbers)
	}
}

func TestGetFixedLines_IdenticalLineIgnored(t *testing.T) {
	reply := "20|blessings.\n10|the Södërby garden hedge."

	fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance)
	if !replyVal {
		t.Fatalf("a line answered with as it was read rejected the batch")
	}
	if len(fixed) != 1 || fixed[0].Number != 10 {
		t.Errorf("%v put right, want only the line that changed", fixed)
	}
}

func TestAnEmptyReplyIsAnAnswerOfNoCorrections(t *testing.T) {
	fixed, _, reply := GetFixedLines(batch(), "", MaxEditDistance)
	if !reply {
		t.Errorf("a batch the proofreader would leave alone failed")
	}
	if fixed != nil {
		t.Errorf("an empty reply put %v right", fixed)
	}
}

func TestTwoLinesOfNoLettersStandNowhereApart(t *testing.T) {
	if apart := EditDistance(" -- ", "!!!"); apart != 0 {
		t.Errorf("marks alone stand %v apart", apart)
	}
}

func TestTheNumberIsTakenFromTheLineHoweverItIsSeparated(t *testing.T) {
	for _, reply := range []string{
		"10|the Södërby garden hedge.",
		"10 the Södërby garden hedge.",
		"10 | the Södërby garden hedge.",
	} {
		fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance)
		if !replyVal {
			t.Errorf("%q failed", reply)
			continue
		}
		if len(fixed) != 1 || fixed[0].Number != 10 || fixed[0].Text != "the Södërby garden hedge." {
			t.Errorf("%q gave back %v", reply, fixed)
		}
	}
}

func TestGetFixedLines_RejectsNumberFusedToText(t *testing.T) {
	reply := "10the Södërby garden hedge."

	if fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance); replyVal {
		t.Errorf("a row whose number is part of a word was taken as an answer, giving %v", fixed)
	}
}

// A separator the reply invented stands where the line begins, and the letters
// either side of it are the same, so the third gate cannot see it.
func TestGetFixedLines_SeparatorPrefixIgnored(t *testing.T) {
	for _, lineText := range []string{
		"10 - the Sodërby gardin hcdge.",
		"10 — the Sodërby gardin hcdge.",
		"10 → the Sodërby gardin hcdge.",
		"10 : the Sodërby gardin hcdge.",
	} {
		fixed, _, reply := GetFixedLines(batch(), lineText, MaxEditDistance)
		if !reply {
			t.Errorf("%q rejected the batch", lineText)
			continue
		}
		if len(fixed) != 0 {
			t.Errorf("%q put %q in front of the line", lineText, fixed[0].Text)
		}
	}
}

// front of it is not all that changed.
func TestALineThatChangedIsACorrectionHoweverItOpens(t *testing.T) {
	fixed, _, reply := GetFixedLines(batch(), "10|— the Södërby garden hedge.", MaxEditDistance)
	if !reply || len(fixed) != 1 {
		t.Fatalf("answered=%v gave %v", reply, fixed)
	}
}

// A line whose own text opens with digits is a row whose number was left out.
func TestGetFixedLines_RejectsRowWhenNumberIsTextDigits(t *testing.T) {
	dated := Batch{Number: 3, Lines: []Line{
		{Number: 1, Text: "1 January 1970 was a Thursday"},
	}}

	// The number is there, and the line is put right after it.
	if fixed, _, reply := GetFixedLines(dated, "1|1 January 1970 was a Thursday, they say", MaxEditDistance); !reply || len(fixed) != 1 {
		t.Fatalf("a row carrying its number failed: answered=%v %v", reply, fixed)
	}

	// The number was left out, and the line's own first word reads as one.
	if fixed, _, replyVal := GetFixedLines(dated, "1 January 1970 was a Thursday, they say", MaxEditDistance); replyVal {
		t.Errorf("a row that ate the line's own digits was taken as an answer: %v", fixed)
	}
}
