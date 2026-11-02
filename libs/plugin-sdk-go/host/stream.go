package host

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
)

// closeStream closes stream and returns the failure to close it.
func closeStream(stream interface{ Close() error }) error {
	if err := stream.Close(); err != nil {
		return fmt.Errorf("close stream: %w", err)
	}
	return nil
}

// followStream sends each message of stream to out, as read converts it, until ctx is done or the
// stream ends, then closes the stream and out. A stream that ends on a failure while ctx is live sends
// fail of that failure last.
func followStream[M, T any](
	ctx context.Context, stream *connect.ServerStreamForClient[M], out chan<- T, read func(*M) T, fail func(error) T,
) {
	defer close(out)
	err := errors.Join(sendMessages(ctx, stream, out, read), closeStream(stream))
	if err == nil || ctx.Err() != nil {
		return
	}
	select {
	case out <- fail(err):
	case <-ctx.Done():
	}
}

// sendMessages sends each message of stream to out until ctx is done or the stream ends, and returns
// the failure the stream ended on.
func sendMessages[M, T any](ctx context.Context, stream *connect.ServerStreamForClient[M], out chan<- T, read func(*M) T) error {
	for stream.Receive() {
		select {
		case out <- read(stream.Msg()):
		case <-ctx.Done():
			return nil
		}
	}
	if err := stream.Err(); err != nil {
		return readError(err)
	}
	return nil
}
