package identity_test

import "testing"

func TestLinkChannel_CodeForTheTwoAccountsIsTheSameOnBothSides(t *testing.T) {
	t.Parallel()
	displayer, scanner := newPair(t).connect(t)

	got, want := displayer.GetCodeFor("OWNER 1", "JOINER 1"), scanner.GetCodeFor("OWNER 1", "JOINER 1")

	if got != want || len(got) != 6 {
		t.Errorf("codes = %q and %q, want the same six digits", got, want)
	}
}

func TestLinkChannel_CodeFromTheLinkAloneIsNotTheCodeOfTheTwoAccounts(t *testing.T) {
	t.Parallel()
	displayer, scanner := newPair(t).connect(t)

	if scanner.Code() != displayer.Code() {
		t.Fatal("the two sides of a link show different codes of the link alone")
	}

	if displayer.GetCodeFor("OWNER 1", "JOINER 1") == displayer.Code() {
		t.Error("the code of the two accounts is the code the link alone gives")
	}
}

func TestLinkChannel_CodeForAnotherAccountOfTheJoinerDiffers(t *testing.T) {
	t.Parallel()
	displayer, scanner := newPair(t).connect(t)

	owned := displayer.GetCodeFor("OWNER 1", "JOINER 1")

	for name, other := range map[string]string{
		"another joiner": displayer.GetCodeFor("OWNER 1", "JOINER 2"),
		"another owner":  displayer.GetCodeFor("OWNER 2", "JOINER 1"),
		"swapped":        displayer.GetCodeFor("JOINER 1", "OWNER 1"),
		"joined":         displayer.GetCodeFor("OWNER 1JOINER 1", ""),
	} {
		if other == owned {
			t.Errorf("%s: the code is the same as for the real accounts", name)
		}
	}
	if scanner.GetCodeFor("OWNER 1", "JOINER 2") == owned {
		t.Error("a joiner with another account shows the code of the real one")
	}
}
