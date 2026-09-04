package webui

import (
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// closed reports whether a channel has been closed, without waiting on it.
func closed(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}

// TestAVaultNothingIsDrawnFromOwesNothing.
func TestAVaultNothingIsDrawnFromOwesNothing(t *testing.T) {
	var pages leaving

	round := pages.ask()
	if !closed(round.written) {
		t.Error("a vault with no page open was owed something")
	}
	if round.standing() {
		t.Error("a vault with no page open raised a question")
	}
}

// TestAPageSaysNothingUntilItIsAsked.
func TestAPageSaysNothingUntilItIsAsked(t *testing.T) {
	var pages leaving

	token, told, done := pages.listen()
	defer done()

	select {
	case asked := <-told:
		t.Errorf("the page was told to flush %q before anything asked", asked)
	default:
	}

	round := pages.ask()
	if closed(round.written) {
		t.Error("a page that has said nothing was counted as having written")
	}
	select {
	case asked := <-told:
		if asked != token {
			t.Errorf("the page was told to flush under %q and listens under %q", asked, token)
		}
	default:
		t.Fatal("the page was never told to flush")
	}

	pages.flushed(token, wrote)
	if !closed(round.written) {
		t.Error("a page that wrote what it owed is still owed")
	}
	if round.standing() {
		t.Error("a page that wrote what it owed raised a question")
	}
}

// TestAQuestionEndsTheRoundAndNotTheWait. A page raising a question has
// answered, so nothing is waited for; and it has not written, so the window has
// no reason to go.
func TestAQuestionEndsTheRoundAndNotTheWait(t *testing.T) {
	var pages leaving

	token, told, done := pages.listen()
	defer done()

	round := pages.ask()
	<-told
	pages.flushed(token, asks)

	if !round.standing() {
		t.Error("a question was raised and the round does not say so")
	}
	if closed(round.written) {
		t.Fatal("a page with a question standing was counted as having written")
	}

	pages.flushed(token, wrote)
	if !closed(round.written) {
		t.Error("the question was answered and the round is still owed something")
	}
}

// TestOneQuestionAmongManyPagesKeepsTheWindow.
func TestOneQuestionAmongManyPagesKeepsTheWindow(t *testing.T) {
	var pages leaving

	first, toldFirst, doneFirst := pages.listen()
	defer doneFirst()
	second, toldSecond, doneSecond := pages.listen()
	defer doneSecond()

	round := pages.ask()
	<-toldFirst
	<-toldSecond
	pages.flushed(first, wrote)
	pages.flushed(second, asks)

	if !round.standing() {
		t.Error("a question was raised and the round does not say so")
	}
	if closed(round.written) {
		t.Fatal("the window went with a question standing")
	}

	pages.flushed(second, wrote)
	if !closed(round.written) {
		t.Error("every page has written and the round is still owed something")
	}
}

// TestASecondRoundAsksEveryPageAgain. What a page said about the round before
// is not an answer to this one.
func TestASecondRoundAsksEveryPageAgain(t *testing.T) {
	var pages leaving

	token, told, done := pages.listen()
	defer done()

	first := pages.ask()
	<-told
	pages.flushed(token, wrote)
	if !closed(first.written) {
		t.Fatal("the page wrote what it owed and the round is still owed something")
	}

	second := pages.ask()
	if !closed(first.over) {
		t.Error("the round before was left running")
	}
	if closed(second.written) {
		t.Error("the second round took the first round's answer for its own")
	}
	select {
	case asked := <-told:
		if asked != token {
			t.Errorf("the page was told to flush under %q and listens under %q", asked, token)
		}
	default:
		t.Fatal("the second round never told the page to flush")
	}

	pages.flushed(token, wrote)
	if !closed(second.written) {
		t.Error("the page wrote what it owed and the second round is still owed it")
	}
}

// TestAPageThatOpensWhileTheWindowIsGoingIsAsked.
func TestAPageThatOpensWhileTheWindowIsGoingIsAsked(t *testing.T) {
	var pages leaving

	round := pages.ask()
	if !closed(round.written) {
		t.Fatal("a vault with no page open was owed something")
	}

	token, told, done := pages.listen()
	defer done()

	select {
	case asked := <-told:
		if asked != token {
			t.Errorf("the page was told to flush under %q and listens under %q", asked, token)
		}
	default:
		t.Error("a page that opened while the window was going was never asked")
	}
}

// TestAPageThatGoesWithNothingStandingStopsBeingOwed.
func TestAPageThatGoesWithNothingStandingStopsBeingOwed(t *testing.T) {
	var pages leaving

	_, _, done := pages.listen()

	round := pages.ask()
	if closed(round.written) {
		t.Fatal("a page that has said nothing was counted as having written")
	}

	done()
	if !closed(round.written) {
		t.Error("a page that stopped listening is still owed")
	}
}

// TestAPageThatGoesWithAQuestionStandingIsStillOwed. The text is in that
// webview's buffer and nowhere else, and the stream ending says nothing about
// where it should end up.
func TestAPageThatGoesWithAQuestionStandingIsStillOwed(t *testing.T) {
	var pages leaving

	token, told, done := pages.listen()
	first := pages.ask()
	<-told
	pages.flushed(token, asks)

	done()

	if closed(first.written) {
		t.Fatal("the window went with the buffer of a page that had a question standing")
	}

	// The client comes back under a token of its own, and is asked afresh.
	second := pages.ask()
	if closed(second.written) {
		t.Fatal("a round with the question still standing was owed nothing")
	}
	again, toldAgain, doneAgain := pages.listen()
	defer doneAgain()
	if again == token {
		t.Fatalf("the page that came back listens under the token that went, %q", again)
	}
	select {
	case asked := <-toldAgain:
		if asked != again {
			t.Errorf("the page was told to flush under %q and listens under %q", asked, again)
		}
	default:
		t.Fatal("the page that came back was never asked")
	}

	pages.flushed(again, asks)
	if !second.standing() {
		t.Error("the page raised its question again and the round does not say so")
	}
	pages.flushed(again, wrote)
	if !closed(second.written) {
		t.Error("the question was answered and the round is still owed something")
	}
}

// TestAPageThatGoesWithAQuestionStandingAndDoesNotComeBackIsSilence. Silence is
// what a round waits its bound for, so the window is not held for ever by a
// webview that will not answer again.
func TestAPageThatGoesWithAQuestionStandingAndDoesNotComeBackIsSilence(t *testing.T) {
	var pages leaving

	token, told, done := pages.listen()
	pages.ask()
	<-told
	pages.flushed(token, asks)
	done()

	round := pages.ask()
	if closed(round.written) {
		t.Error("a round took a page that went with a question standing as written")
	}
	if round.standing() {
		t.Error("a page that is no longer there was counted as raising a question")
	}

	// A page that says nothing is what the bound is for, and nothing here ends
	// the round before it.
	select {
	case <-round.written:
		t.Error("the round ended without the page saying anything")
	case <-time.After(50 * time.Millisecond):
	}
}

// TestAPageWithNothingLeftAndOneThatHasWrittenBothLetTheWindowGo.
func TestAPageWithNothingLeftAndOneThatHasWrittenBothLetTheWindowGo(t *testing.T) {
	for name, said := range map[string]owed{
		"nothing owed":       left(v1.Owed_OWED_NOTHING),
		"everything written": left(v1.Owed_OWED_WRITTEN),
	} {
		t.Run(name, func(t *testing.T) {
			var pages leaving

			token, told, done := pages.listen()
			defer done()

			round := pages.ask()
			<-told
			pages.flushed(token, said)

			if !closed(round.written) {
				t.Error("the window was held by a page that had nothing left")
			}
		})
	}
}
