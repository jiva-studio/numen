package wire

import (
	"context"
	"sync"
	"time"

	"connectrpc.com/connect"
)

// RunWatch runs a watch over a server stream until run returns, the client is gone or ctx ends. run
// sends through the send it is given, and returns nil when its source is over.
//
// The context of a request ends when the window closes, not when a page reloads, so a gone client is
// noticed only through a failed write. While run sends nothing for interval, RunWatch sends the message
// makeKeepalive returns, given the last message sent, or an empty message when makeKeepalive is nil. A failed
// keepalive ends the watch with that failure. No write reaches the stream once RunWatch has returned.
func RunWatch[Res any](
	ctx context.Context,
	out *connect.ServerStream[Res],
	interval time.Duration,
	makeKeepalive func(last *Res) *Res,
	run func(ctx context.Context, send func(*Res) error) error,
) error {
	watching, stop := context.WithCancel(ctx)
	var (
		mu      sync.Mutex
		isOver  bool
		hasSent bool
		last    *Res
		gone    error
		beating sync.WaitGroup
	)
	defer func() {
		stop()
		mu.Lock()
		isOver = true
		mu.Unlock()
		beating.Wait()
	}()
	send := func(message *Res) error {
		mu.Lock()
		defer mu.Unlock()
		if isOver {
			return context.Canceled
		}
		if err := out.Send(message); err != nil {
			return err
		}
		last, hasSent = message, true
		return nil
	}
	beating.Go(func() {
		repeat := time.NewTicker(interval)
		defer repeat.Stop()
		for {
			select {
			case <-watching.Done():
				return
			case <-repeat.C:
				if err := sendKeepalive(&mu, &hasSent, &last, makeKeepalive, out.Send); err != nil {
					mu.Lock()
					gone = err
					mu.Unlock()
					stop()
					return
				}
			}
		}
	})
	err := run(watching, send)
	mu.Lock()
	defer mu.Unlock()
	if gone != nil {
		return gone
	}
	return err
}

func sendKeepalive[Res any](
	mu *sync.Mutex, hasSent *bool, last **Res, makeKeepalive func(*Res) *Res, send func(*Res) error,
) error {
	mu.Lock()
	defer mu.Unlock()
	if *hasSent {
		*hasSent = false
		return nil
	}
	if makeKeepalive == nil {
		return send(new(Res))
	}
	return send(makeKeepalive(*last))
}

// GetLastOrEmpty is the keepalive of a watch whose every message holds the whole state: the last message
// again, or an empty one where none was sent.
func GetLastOrEmpty[Res any](last *Res) *Res {
	if last == nil {
		return new(Res)
	}
	return last
}
