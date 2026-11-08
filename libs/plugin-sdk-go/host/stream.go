package host

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
)

func closeStream(stream interface{ Close() error }) error {
	if err := stream.Close(); err != nil {
		return fmt.Errorf("close stream: %w", err)
	}
	return nil
}

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
