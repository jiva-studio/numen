package fakeserver_test

import (
	"bytes"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

type stepClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *stepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *stepClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newRelayOn(tb testing.TB) (*syncclient.LinkClient, *stepClock) {
	tb.Helper()
	clock := &stepClock{now: time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)}
	server := fakeserver.New(fakeserver.Config{Now: clock.Now})
	address := serve(tb, server)
	return syncclient.NewLinkClient(address, http.DefaultClient), clock
}

func newChannel(tb testing.TB, relay *syncclient.LinkClient) []byte {
	tb.Helper()
	channel, err := relay.CreateChannel(tb.Context())
	if err != nil {
		tb.Fatalf("CreateChannel() = %v", err)
	}
	return channel.ID
}

func TestCreateChannel_MakesASixteenByteIdThatLivesTenMinutes(t *testing.T) {
	t.Parallel()
	relay, clock := newRelayOn(t)

	channel, err := relay.CreateChannel(t.Context())

	if err != nil || len(channel.ID) != 16 {
		t.Fatalf("channel = %+v, %v, want a 16 byte id", channel, err)
	}
	if want := clock.Now().Add(10 * time.Minute); !channel.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", channel.ExpiresAt, want)
	}
	other, err := relay.CreateChannel(t.Context())
	if err != nil || bytes.Equal(other.ID, channel.ID) {
		t.Errorf("a second channel has the same id: %v", err)
	}
}

func TestSendMessage_DeliversInOrderToTheSideItIsAddressedTo(t *testing.T) {
	t.Parallel()
	relay, _ := newRelayOn(t)
	id := newChannel(t, relay)
	for i, post := range []struct {
		to   syncclient.LinkSide
		body string
	}{{syncclient.LinkSideDisplayer, "first"}, {syncclient.LinkSideScanner, "second"}, {syncclient.LinkSideDisplayer, "third"}} {
		seq, err := relay.SendMessage(t.Context(), id, post.to, []byte(post.body))
		if err != nil || seq != uint32(i+1) {
			t.Fatalf("SendMessage(%q) = %d, %v, want seq %d", post.body, seq, err, i+1)
		}
	}

	forDisplayer, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideDisplayer, 0, 0)
	if err != nil || len(forDisplayer.Messages) != 2 || string(forDisplayer.Messages[0].Body) != "first" || forDisplayer.Messages[1].Seq != 3 {
		t.Errorf("displayer reads %+v, %v, want first (seq 1) and third (seq 3)", forDisplayer, err)
	}
	forScanner, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 0, 0)
	if err != nil || len(forScanner.Messages) != 1 || string(forScanner.Messages[0].Body) != "second" {
		t.Errorf("scanner reads %+v, %v, want second", forScanner, err)
	}
	after, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideDisplayer, 1, 0)
	if err != nil || len(after.Messages) != 1 || string(after.Messages[0].Body) != "third" {
		t.Errorf("displayer after seq 1 reads %+v, %v, want third", after, err)
	}
}

func TestWaitMessages_AnswersWhenAMessageIsPostedWhileItWaits(t *testing.T) {
	t.Parallel()
	relay, _ := newRelayOn(t)
	id := newChannel(t, relay)
	answer := make(chan syncclient.LinkMessages, 1)
	go func() {
		got, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 0, 10*time.Second)
		if err != nil {
			t.Error(err)
		}
		answer <- got
	}()
	var posted bool
	for range 50 {
		select {
		case got := <-answer:
			if len(got.Messages) != 1 || string(got.Messages[0].Body) != "hello" {
				t.Errorf("the waiter got %+v, want hello", got)
			}
			return
		case <-time.After(100 * time.Millisecond):
			if !posted {
				if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("hello")); err != nil {
					t.Fatal(err)
				}
				posted = true
			}
		}
	}
	t.Fatal("the waiter did not answer")
}

func TestWaitMessages_RefusesASecondWaitOnTheSameSide(t *testing.T) {
	t.Parallel()
	relay, _ := newRelayOn(t)
	id := newChannel(t, relay)
	type answer struct {
		messages syncclient.LinkMessages
		err      error
	}
	answers := make(chan answer, 2)
	for range 2 {
		go func() {
			messages, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 0, 5*time.Second)
			answers <- answer{messages: messages, err: err}
		}()
	}
	refused := <-answers
	if !errors.Is(refused.err, syncprotocol.ErrLinkBusy) {
		t.Errorf("second wait: error = %v, want ErrLinkBusy", refused.err)
	}
	if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("end")); err != nil {
		t.Fatal(err)
	}
	kept := <-answers
	if kept.err != nil || len(kept.messages.Messages) != 1 || string(kept.messages.Messages[0].Body) != "end" {
		t.Errorf("the first wait = %+v, %v, want it kept until end", kept.messages, kept.err)
	}
}

func TestCloseChannel_EndsTheChannelAndTellsTheWaitingPeerOnce(t *testing.T) {
	t.Parallel()
	relay, _ := newRelayOn(t)
	id := newChannel(t, relay)
	if err := relay.CloseChannel(t.Context(), id); err != nil {
		t.Fatalf("CloseChannel() = %v", err)
	}
	told, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 0, 0)
	if err != nil || !told.IsClosed {
		t.Errorf("the peer reads %+v, %v, want is_closed", told, err)
	}
	if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("late")); !errors.Is(err, syncprotocol.ErrChannelGone) {
		t.Errorf("post to a closed channel: error = %v, want ErrChannelGone", err)
	}
	if _, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 0, 0); !errors.Is(err, syncprotocol.ErrChannelGone) {
		t.Errorf("a second wait on a closed channel: error = %v, want ErrChannelGone", err)
	}
	if err := relay.CloseChannel(t.Context(), id); !errors.Is(err, syncprotocol.ErrChannelGone) {
		t.Errorf("a second close: error = %v, want ErrChannelGone", err)
	}
}

func TestSendMessage_TheTwentyFourthMessageClosesTheChannel(t *testing.T) {
	t.Parallel()
	relay, _ := newRelayOn(t)
	id := newChannel(t, relay)
	for i := range 24 {
		if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte{byte(i + 1)}); err != nil {
			t.Fatalf("message %d: %v", i+1, err)
		}
	}
	if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("25")); !errors.Is(err, syncprotocol.ErrChannelGone) {
		t.Errorf("the 25th message: error = %v, want ErrChannelGone", err)
	}
	last, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 23, 0)
	if err != nil || len(last.Messages) != 1 || !last.IsClosed {
		t.Errorf("reader after seq 23 = %+v, %v, want the 24th message and is_closed", last, err)
	}
}

func TestLinkChannel_ExpiresTenMinutesAfterCreation(t *testing.T) {
	t.Parallel()
	relay, clock := newRelayOn(t)
	id := newChannel(t, relay)
	clock.Advance(10*time.Minute - time.Second)
	if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("in time")); err != nil {
		t.Fatalf("a post before the end: %v", err)
	}
	clock.Advance(2 * time.Second)
	if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, []byte("late")); !errors.Is(err, syncprotocol.ErrChannelGone) {
		t.Errorf("a post after the end: error = %v, want ErrChannelGone", err)
	}
	if _, err := relay.WaitMessages(t.Context(), id, syncclient.LinkSideScanner, 0, 0); !errors.Is(err, syncprotocol.ErrChannelGone) {
		t.Errorf("a wait after the end: error = %v, want ErrChannelGone", err)
	}
}

func TestLinkChannel_UnknownIdIsExpiredForEveryCallWithNoDetail(t *testing.T) {
	t.Parallel()
	relay, _ := newRelayOn(t)
	unknown := bytes.Repeat([]byte{7}, 16)
	_, post := relay.SendMessage(t.Context(), unknown, syncclient.LinkSideScanner, []byte("x"))
	_, wait := relay.WaitMessages(t.Context(), unknown, syncclient.LinkSideScanner, 0, 0)
	closing := relay.CloseChannel(t.Context(), unknown)
	for name, err := range map[string]error{"post": post, "wait": wait, "close": closing} {
		if !errors.Is(err, syncprotocol.ErrChannelGone) || connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("%s: error = %v, want ErrChannelGone from NOT_FOUND", name, err)
		}
		if failure, ok := errors.AsType[*connect.Error](err); ok && len(failure.Details()) != 0 {
			t.Errorf("%s: the refusal carries details", name)
		}
	}
}

func TestSendMessage_RefusesAnEmptyBodyAndABodyOverTheLimit(t *testing.T) {
	t.Parallel()
	relay, _ := newRelayOn(t)
	id := newChannel(t, relay)
	for name, body := range map[string][]byte{"empty": nil, "over the limit": make([]byte, syncprotocol.MaxLinkMessage+1)} {
		_, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, body)
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%s: error = %v, want INVALID_ARGUMENT", name, err)
		}
	}
	if _, err := relay.SendMessage(t.Context(), id, syncclient.LinkSideScanner, make([]byte, syncprotocol.MaxLinkMessage)); err != nil {
		t.Errorf("a body of the limit: %v", err)
	}
}

func TestCreateChannel_NeedsNoSession(t *testing.T) {
	t.Parallel()
	server := fakeserver.New(fakeserver.Config{})
	relay := syncclient.NewLinkClient(serve(t, server), http.DefaultClient)
	if _, err := relay.CreateChannel(t.Context()); err != nil {
		t.Errorf("CreateChannel() without a sign-in = %v", err)
	}
}
