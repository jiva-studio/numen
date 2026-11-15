package numbered

import (
	"testing"
)

func batch() Batch {
	return Batch{Number: 7, Lines: []Line{
		{Number: 10, Text: "the Sodërby gardin hcdge. "},
		{Number: 11, Text: "Sodërby Tradgard's Ordbok "},
		{Number: 12, Text: "TRADGARD ORDBOK "},
		{Number: 13, Text: "The hedge will very soon need cutting back "},
		{Number: 20, Text: "blessings."},
	}}
}

func TestGetFixedLines_RejectsReplyContainingMarker(t *testing.T) {
	for _, reply := range []string{
		"10|the " + Opens + "12Södërby garden hedge.",
		"10|the 12" + Closes + "Södërby garden hedge.",
	} {
		fixed, hasOverflow, replyVal := GetFixedLines(batch(), reply, MaxEditDistance)
		if replyVal || hasOverflow || fixed != nil {
			t.Errorf("reply %q gave fixed=%v hasOverflow=%v ok=%v", reply, fixed, hasOverflow, replyVal)
		}
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
		{"Sodërby Tradgard's Ordbok", "Södërby Trädgård's Ordbök", 0, 0, true},
		// A heading answered with the paragraph under it.
		{"TRADGARD ORDBOK", "TRÄDGÅRD ORDBÖK At the request of the Sodërby committee", 0.60, 0.75, false},
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
		"11|Södërby Trädgård's Ordbök\n" +
		"12|TRÄDGÅRD ORDBÖK At the request of the Sodërby committee\n" +
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
	if apart := EditDistance("$€+", "£¥="); apart != 0 {
		t.Errorf("symbols alone stand %v apart", apart)
	}
	if apart := EditDistance("a", "ab"); apart != 0.5 {
		t.Errorf("a -> ab stands %v apart, want 0.5", apart)
	}
}

func TestTheNumberIsTakenFromTheLineHoweverItIsSeparated(t *testing.T) {
	for _, reply := range []string{
		"10|the Södërby garden hedge.",
		"10 the Södërby garden hedge.",
		"10\tthe Södërby garden hedge.",
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
	for _, reply := range []string{
		"10the Södërby garden hedge.",
		"10x|the Södërby garden hedge.",
	} {
		if fixed, _, replyVal := GetFixedLines(batch(), reply, MaxEditDistance); replyVal {
			t.Errorf("a row whose number is part of a word was taken as an answer, giving %v", fixed)
		}
	}
}

// A separator that the reply adds at the start of the line leaves the letters
// on either side unchanged, so the third gate cannot detect it.
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

func TestGateRowHelpers(t *testing.T) {
	withZero := Batch{
		Number:     1,
		IsJoinable: false,
		Lines: []Line{
			{Number: 0, Text: "zero"},
			{Number: 10, Text: "alpha"},
			{Number: 11, Text: "beta"},
		},
	}
	read := indexBatchLines(withZero)
	if line, keep, reaches, ok := processRow("bad", withZero, read, map[int]bool{}, 0); line != (Line{}) || keep || reaches || ok {
		t.Errorf("non-numbered row gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}
	if line, keep, reaches, ok := processRow("10-11|joined", withZero, read, map[int]bool{}, 0); line != (Line{}) || keep || reaches || ok {
		t.Errorf("range on non-joinable batch gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}
	if line, keep, reaches, ok := processRow("99|missing", withZero, read, map[int]bool{}, 0); line != (Line{}) || keep || reaches || ok {
		t.Errorf("unknown line gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}
	joinBatch := withZero
	joinBatch.IsJoinable = true
	if line, keep, reaches, ok := processRow("10-12|overflow", joinBatch, read, map[int]bool{}, 0); line != (Line{}) || keep || !reaches || !ok {
		t.Errorf("overflow range gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}
	if line, keep, reaches, ok := processRow("10|ALPHA", withZero, read, map[int]bool{10: true}, 0); line != (Line{}) || keep || reaches || ok {
		t.Errorf("already claimed line gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}
	if line, keep, reaches, ok := processRow("10|...", withZero, read, map[int]bool{}, 0); line != (Line{}) || keep || reaches || ok {
		t.Errorf("emptied letters gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}
	if line, keep, reaches, ok := processRow("10|alpha", withZero, read, map[int]bool{}, 0); line != (Line{}) || keep || reaches || !ok {
		t.Errorf("unchanged line gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}
	if line, keep, reaches, ok := processRow("10|alphe", withZero, read, map[int]bool{}, 0); line != (Line{Number: 10, Last: 10, Text: "alphe"}) || !keep || reaches || !ok {
		t.Errorf("valid change gave %+v keep=%v reaches=%v ok=%v", line, keep, reaches, ok)
	}

	if joined, reaches, ok := matchLines(read, 99, 99, 11); joined != "" || reaches || ok {
		t.Errorf("matchLines missing start: %q %v %v", joined, reaches, ok)
	}
	if joined, reaches, ok := matchLines(map[int]string{1: "a"}, 1, 2, 2); joined != "" || reaches || ok {
		t.Errorf("matchLines missing at last: %q %v %v", joined, reaches, ok)
	}
	if joined, reaches, ok := matchLines(map[int]string{1: "a"}, 1, 2, 1); joined != "" || !reaches || !ok {
		t.Errorf("matchLines reaching past last: %q %v %v", joined, reaches, ok)
	}
	if joined, reaches, ok := matchLines(map[int]string{1: "hello", 2: "world"}, 1, 2, 2); joined != "hello world" || reaches || !ok {
		t.Errorf("matchLines joined: %q %v %v", joined, reaches, ok)
	}

	if !isValidRow("1|!", "---", "!", true) || isValidRow("1|!", "a", "!", true) {
		t.Error("isValidRow mishandled single-letter and letterless lines")
	}
	if canKeepCorrection("", "", 1, 1, 0) || !canKeepCorrection("abcd", "abce", 1, 1, 0.25) || canKeepCorrection("abcd", "abce", 1, 1, 0.24) {
		t.Error("canKeepCorrection mishandled empty or boundary distance")
	}
}

func TestReadRowAndFenceHelpers(t *testing.T) {
	for _, bad := range []string{
		"",
		"123",
		"abc",
		"999999999999999999999999999999|overflow",
		"5-5|same",
		"5-3|backwards",
		"5-6",
		"5-|missing",
		"5-999999999999999999999999999999|overflow",
		"10x|fused",
	} {
		if s, e, txt, barred, ok := readRow(bad); s != 0 || e != 0 || txt != "" || barred || ok {
			t.Errorf("readRow(%q) = %d, %d, %q, %v, %v", bad, s, e, txt, barred, ok)
		}
	}
	if endLine, rem, ok := readRowRange("-5|same", 5); endLine != 0 || rem != "" || ok {
		t.Errorf("readRowRange(-5|same, 5) = %d, %q, %v", endLine, rem, ok)
	}
	if endLine, rem, ok := readRowRange("-6", 5); endLine != 0 || rem != "" || ok {
		t.Errorf("readRowRange(-6, 5) = %d, %q, %v", endLine, rem, ok)
	}
	if got := readLeadingDigits(""); got != "" {
		t.Errorf("readLeadingDigits empty = %q", got)
	}
	if got := readLeadingDigits("abc"); got != "" {
		t.Errorf("readLeadingDigits non-digits = %q", got)
	}
	if got := readLeadingDigits("42"); got != "42" {
		t.Errorf("readLeadingDigits all digits = %q", got)
	}
	if isPrefixedBySymbols("", "---") ||
		isPrefixedBySymbols("hello", "-a hello") ||
		isPrefixedBySymbols("hello", "a hello") ||
		isPrefixedBySymbols("hello", "1 hello") ||
		!isPrefixedBySymbols("hello", "- hello") {
		t.Error("isPrefixedBySymbols failed")
	}
	if got := stripFence("```text"); got != "" {
		t.Errorf("stripFence single line = %q", got)
	}
	if got := stripFence("```\n```"); got != "" {
		t.Errorf("stripFence empty fenced block = %q", got)
	}
	if got := stripFence("```\n10|hello"); got != "10|hello" {
		t.Errorf("stripFence unclosed block = %q", got)
	}
}
