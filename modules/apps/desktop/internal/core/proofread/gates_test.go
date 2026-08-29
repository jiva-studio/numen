package proofread_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/proofread"
)

// page is a page as it was read, misreadings and all.
func page() proofread.Page {
	return proofread.Page{At: 7, Lines: []proofread.Line{
		{At: 10, Text: "the Sodërby gardin hcdge. "},
		{At: 11, Text: "Sodërby Tradgard's Handbok "},
		{At: 12, Text: "TRADGARD HANDBOK "},
		{At: 13, Text: "The hedge will very soon need cutting back "},
		{At: 20, Text: "blessings."},
	}}
}

func TestAMarkOfOursComingBackRefusesThePage(t *testing.T) {
	reply := "10|the " + proofread.Opens + "12" + proofread.Closes + "Södërby garden hedge."

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if answered {
		t.Errorf("a reply carrying %s was taken as an answer", proofread.Opens)
	}
	if fixed != nil {
		t.Errorf("a refused page gave back %v", fixed)
	}
}

func TestALineThePageDidNotNameRefusesThePage(t *testing.T) {
	reply := "10|the Södërby garden hedge.\n999|a line this page never printed"

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if answered {
		t.Errorf("a reply naming line 999 was taken as an answer")
	}
	if fixed != nil {
		t.Errorf("a refused page gave back %v", fixed)
	}
}

func TestAReplyLineWithoutANumberAndABarRefusesThePage(t *testing.T) {
	reply := "Here are the lines I would put right:\n10|the Södërby garden hedge."

	if fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart); answered {
		t.Errorf("a reply that talks was taken as an answer, giving %v", fixed)
	}
}

func TestAFencedReplyIsUnwrapped(t *testing.T) {
	reply := "```text\n10|the Södërby garden hedge.\n20|blessings!\n```"

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if !answered {
		t.Fatalf("a fenced reply was refused")
	}
	if len(fixed) != 2 {
		t.Fatalf("%d lines out of a fenced reply, want two: %v", len(fixed), fixed)
	}
	if fixed[0].At != 10 || fixed[0].Text != "the Södërby garden hedge." {
		t.Errorf("line %d says %q", fixed[0].At, fixed[0].Text)
	}
}

func TestTheLettersOfACorrectionMayOnlyMoveSoFar(t *testing.T) {
	for _, one := range []struct {
		was, put    string
		least, most float64
		kept        bool
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
		apart := proofread.Apart(one.was, one.put)
		if apart < one.least || apart > one.most {
			t.Errorf("%q -> %q stands %.4f apart, want between %v and %v", one.was, one.put, apart, one.least, one.most)
		}
		if kept := apart <= proofread.LettersApart; kept != one.kept {
			t.Errorf("%q -> %q stands %.4f apart, kept %v, want kept %v", one.was, one.put, apart, kept, one.kept)
		}
	}
}

func TestACorrectionThatMovedTooFarIsDroppedAndThePageKept(t *testing.T) {
	reply := "10|the Södërby garden hedge.\n" +
		"11|Södërby Trädgård's Handbök\n" +
		"12|TRÄDGÅRD HANDBÖK At the request of the Sodërby committee\n" +
		"13|hedge"

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if !answered {
		t.Fatalf("one correction that moved too far refused the whole page")
	}
	var numbers []int
	for _, line := range fixed {
		numbers = append(numbers, line.At)
	}
	if len(numbers) != 2 || numbers[0] != 10 || numbers[1] != 11 {
		t.Errorf("lines %v put right, want the two whose letters stayed put", numbers)
	}
}

func TestACorrectionSayingWhatTheLineSaysIsNotACorrection(t *testing.T) {
	reply := "20|blessings.\n10|the Södërby garden hedge."

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if !answered {
		t.Fatalf("a line answered with as it was read refused the page")
	}
	if len(fixed) != 1 || fixed[0].At != 10 {
		t.Errorf("%v put right, want only the line that changed", fixed)
	}
}

func TestAnEmptyReplyIsAnAnswerOfNoCorrections(t *testing.T) {
	fixed, answered := proofread.Fixed(page(), "", proofread.LettersApart)
	if !answered {
		t.Errorf("a page the proofreader would leave alone was refused")
	}
	if fixed != nil {
		t.Errorf("an empty reply put %v right", fixed)
	}
}

func TestTwoLinesOfNoLettersStandNowhereApart(t *testing.T) {
	if apart := proofread.Apart(" -- ", "!!!"); apart != 0 {
		t.Errorf("marks alone stand %v apart", apart)
	}
}

func TestTheNumberIsTakenFromTheLineHoweverItIsSeparated(t *testing.T) {
	for _, reply := range []string{
		"10|the Södërby garden hedge.",
		"10 the Södërby garden hedge.",
		"10 | the Södërby garden hedge.",
	} {
		fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
		if !answered {
			t.Errorf("%q was refused", reply)
			continue
		}
		if len(fixed) != 1 || fixed[0].At != 10 || fixed[0].Text != "the Södërby garden hedge." {
			t.Errorf("%q gave back %v", reply, fixed)
		}
	}
}

func TestANumberRunningIntoTheLineRefusesThePage(t *testing.T) {
	reply := "10the Södërby garden hedge."

	if fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart); answered {
		t.Errorf("a row whose number is part of a word was taken as an answer, giving %v", fixed)
	}
}

// A separator the reply invented stands where the line begins, and the letters
// either side of it are the same, so the third gate cannot see it.
func TestAWordlessThingPutInFrontOfALineIsNoCorrection(t *testing.T) {
	for _, put := range []string{
		"10 - the Sodërby gardin hcdge.",
		"10 — the Sodërby gardin hcdge.",
		"10 → the Sodërby gardin hcdge.",
		"10 : the Sodërby gardin hcdge.",
	} {
		fixed, answered := proofread.Fixed(page(), put, proofread.LettersApart)
		if !answered {
			t.Errorf("%q refused the page", put)
			continue
		}
		if len(fixed) != 0 {
			t.Errorf("%q put %q in front of the line", put, fixed[0].Text)
		}
	}
}

// The same line, corrected as well as fronted, is a correction: what stands in
// front of it is not all that changed.
func TestALineThatChangedIsACorrectionHoweverItOpens(t *testing.T) {
	fixed, answered := proofread.Fixed(page(), "10|— the Södërby garden hedge.", proofread.LettersApart)
	if !answered || len(fixed) != 1 {
		t.Fatalf("answered=%v gave %v", answered, fixed)
	}
}

// A line whose own text opens with digits is a row whose number was left out.
func TestARowWhoseNumberIsTheLinesOwnDigitsRefusesThePage(t *testing.T) {
	dated := proofread.Page{At: 3, Lines: []proofread.Line{
		{At: 1, Text: "1 January 1970 was a Thursday"},
	}}

	// The number is there, and the line is put right after it.
	if fixed, answered := proofread.Fixed(dated, "1|1 January 1970 was a Thursday, they say", proofread.LettersApart); !answered || len(fixed) != 1 {
		t.Fatalf("a row carrying its number was refused: answered=%v %v", answered, fixed)
	}

	// The number was left out, and the line's own first word reads as one.
	if fixed, answered := proofread.Fixed(dated, "1 January 1970 was a Thursday, they say", proofread.LettersApart); answered {
		t.Errorf("a row that ate the line's own digits was taken as an answer: %v", fixed)
	}
}
