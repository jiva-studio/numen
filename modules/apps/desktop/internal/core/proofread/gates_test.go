package proofread_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/proofread"
)

// page is a page as it was read, misreadings and all.
func page() proofread.Page {
	return proofread.Page{At: 7, Lines: []proofread.Line{
		{At: 10, Text: "the Gundicā teinple. "},
		{At: 11, Text: "Śrila Jagadiśa Pandita's Sripat "},
		{At: 12, Text: "VAISNAVA MANJUSA "},
		{At: 13, Text: "Supreme Lord Krsna will very soon bestow His "},
		{At: 20, Text: "blessings."},
	}}
}

func TestAMarkOfOursComingBackRefusesThePage(t *testing.T) {
	reply := "10|the " + proofread.Opens + "12" + proofread.Closes + "Guṇḍicā temple."

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if answered {
		t.Errorf("a reply carrying %s was taken as an answer", proofread.Opens)
	}
	if fixed != nil {
		t.Errorf("a refused page gave back %v", fixed)
	}
}

func TestALineThePageDidNotNameRefusesThePage(t *testing.T) {
	reply := "10|the Guṇḍicā temple.\n999|a line this page never printed"

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if answered {
		t.Errorf("a reply naming line 999 was taken as an answer")
	}
	if fixed != nil {
		t.Errorf("a refused page gave back %v", fixed)
	}
}

func TestAReplyLineWithoutANumberAndABarRefusesThePage(t *testing.T) {
	reply := "Here are the lines I would put right:\n10|the Guṇḍicā temple."

	if fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart); answered {
		t.Errorf("a reply that talks was taken as an answer, giving %v", fixed)
	}
}

func TestAFencedReplyIsUnwrapped(t *testing.T) {
	reply := "```text\n10|the Guṇḍicā temple.\n20|blessings!\n```"

	fixed, answered := proofread.Fixed(page(), reply, proofread.LettersApart)
	if !answered {
		t.Fatalf("a fenced reply was refused")
	}
	if len(fixed) != 2 {
		t.Fatalf("%d lines out of a fenced reply, want two: %v", len(fixed), fixed)
	}
	if fixed[0].At != 10 || fixed[0].Text != "the Guṇḍicā temple." {
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
		{"the Gundicā teinple.", "the Guṇḍicā temple.", 0.05, 0.15, true},
		// Diacritics alone: folded, the two say the same letters.
		{"Śrila Jagadiśa Pandita's Sripat", "Śrīla Jagadīśa Paṇḍita's Śrīpat", 0, 0, true},
		// A heading answered with the paragraph under it.
		{"VAISNAVA MANJUSA", "VAIṢṆAVA MAÑJŪṢĀ At the request of Śrila Bhaktivinoda", 0.60, 0.75, false},
		// A line answered with one word of it.
		{"Supreme Lord Krsna will very soon bestow His", "Kṛṣṇa", 0.80, 0.95, false},
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
	reply := "10|the Guṇḍicā temple.\n" +
		"11|Śrīla Jagadīśa Paṇḍita's Śrīpat\n" +
		"12|VAIṢṆAVA MAÑJŪṢĀ At the request of Śrila Bhaktivinoda\n" +
		"13|Kṛṣṇa"

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
	reply := "20|blessings.\n10|the Guṇḍicā temple."

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
