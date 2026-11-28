package cloud

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Watch opens the stream of change hints. An open the server refuses for want of a session signs in again once.
func (c *Cloud) Watch(ctx context.Context) (ports.Watch, error) {
	stream, err := call(ctx, c, func() (*syncclient.WatchStream, error) { return c.watch.WatchChanges(ctx) })
	if err != nil {
		return nil, err
	}
	return newWatch(stream), nil
}

type watchResult struct {
	event ports.HintEvent
	err   error
}

// watch reads the stream in a goroutine of its own, so that Receive can end with its context.
type watch struct {
	stream    *syncclient.WatchStream
	events    chan watchResult
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func newWatch(stream *syncclient.WatchStream) *watch {
	w := &watch{stream: stream, events: make(chan watchResult), stop: make(chan struct{}), done: make(chan struct{}), closeOnce: sync.Once{}}
	go w.forwardEvents()
	return w
}

func (w *watch) forwardEvents() {
	defer close(w.done)
	defer close(w.events)
	for {
		message, err := w.stream.Receive()
		result := watchResult{event: ports.HintEvent{RemoteVaultID: "", Seq: 0}, err: err}
		if err == nil {
			var isEvent bool
			if result.event, isEvent = readEvent(message); !isEvent {
				continue
			}
		}
		select {
		case w.events <- result:
		case <-w.stop:
			return
		}
		if err != nil {
			return
		}
	}
}

func readEvent(message *syncv1.WatchChangesResponse) (ports.HintEvent, bool) {
	if hint := message.GetHint(); hint != nil && len(hint.GetVaultId()) > 0 {
		return ports.HintEvent{RemoteVaultID: hex.EncodeToString(hint.GetVaultId()), Seq: hint.GetSeq()}, true
	}
	return ports.HintEvent{RemoteVaultID: "", Seq: 0}, message.GetHeartbeat() != nil
}

// GetHeartbeat returns the longest silence the server announced.
func (w *watch) GetHeartbeat() time.Duration { return w.stream.Heartbeat() }

// Receive returns the next hint or heartbeat, io.EOF at a clean end, or the error of ctx when it ends first.
func (w *watch) Receive(ctx context.Context) (ports.HintEvent, error) {
	if err := ctx.Err(); err != nil {
		return ports.HintEvent{RemoteVaultID: "", Seq: 0}, fmt.Errorf("receive a hint: %w", err)
	}
	select {
	case result, isOpen := <-w.events:
		if !isOpen {
			return ports.HintEvent{RemoteVaultID: "", Seq: 0}, io.EOF
		}
		return result.event, result.err
	case <-ctx.Done():
		return ports.HintEvent{RemoteVaultID: "", Seq: 0}, fmt.Errorf("receive a hint: %w", ctx.Err())
	}
}

// Close ends the stream and waits for its reader.
func (w *watch) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.stop)
		err = w.stream.Close()
		<-w.done
	})
	return err
}
