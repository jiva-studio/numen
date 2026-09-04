package source

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// whenIdle is what a run calls where it has found the line empty and is about
// to stop the running. It stands beside the running and is set under the lock
// the running is kept under.
func (t *Transcribing) whenIdle(idle func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.idle = idle
}

// A recording named at the instant drainAndRelease finds the line empty is
// transcribed. The run stops the running under the same lock it found the line
// empty under, so the ask that follows is told it began and transcribes the
// recording itself.
//
// A recording told it was queued with no run to transcribe it is a recording
// never transcribed, and a person left waiting on a count that never falls.
//
// The run that ended stops nothing but itself. Its end stands after it let the
// lock go, by which time the run it handed the line to is the one running, and
// a third recording named then is told it waits its turn.
func TestARecordingNamedAsDrainAndReleaseEndsIsTranscribed(t *testing.T) {
	by := &deaf{why: errors.New("not a container anything here decodes")}
	listening, v := listens(t, by, "talks/a.mp3", "talks/b.mp3", "talks/c.mp3")

	// The first transcription is held until this test lets it end, and every
	// one after it until this test is over.
	first, rest := make(chan struct{}), make(chan struct{})
	releaseRest := sync.OnceFunc(func() { close(rest) })
	defer func() {
		releaseRest()
		listening.Wait()
	}()

	var (
		mu     sync.Mutex
		opened int
	)
	hearing := make(chan int, 8)
	listening.with.Open = func(context.Context, func(string, int64, int64)) (port.Transcriber, func() error, error) {
		mu.Lock()
		opened++
		n := opened
		mu.Unlock()
		hearing <- n
		if n == 1 {
			<-first
		} else {
			<-rest
		}
		return by, by.Close, nil
	}

	crossed := newInstant()
	listening.whenIdle(func() {
		crossed.at(func() port.Taking { return listening.Start(v, "talks/b.mp3") })
	})

	if got := listening.Start(v, "talks/a.mp3"); got != port.Began {
		t.Fatalf("the first recording was not transcribed: %v", got)
	}
	if n := begun(t, hearing); n != 1 {
		t.Fatalf("the first transcription is the %dth", n)
	}
	close(first)

	if taken := crossed.answered(t); taken != port.Began {
		t.Fatalf("the recording named as the line emptied was told %v", taken)
	}
	if n := begun(t, hearing); n != 2 {
		t.Fatalf("the recording named as the line emptied is the %dth transcription", n)
	}

	// The run that emptied the line ends now. Its end stands anywhere after it
	// let the lock go, and it stops nothing but itself: the run it handed the
	// line to is the one running, and a third recording waits its turn.
	listening.release(1)
	if got := listening.Start(v, "talks/c.mp3"); got != port.Queued {
		t.Errorf("a recording named while one was being transcribed was told %v", got)
	}

	releaseRest()
	listening.Wait()

	mu.Lock()
	heard := opened
	mu.Unlock()
	if heard != 3 {
		t.Errorf("a transcriber was opened %d times", heard)
	}
	if listening.Waiting() != 0 {
		t.Errorf("%d recordings were left in line", listening.Waiting())
	}
}
