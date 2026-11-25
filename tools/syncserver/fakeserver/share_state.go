package fakeserver

import (
	"errors"
	"time"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	shareURLsPerHour = 1000
	shareWindow      = time.Hour
)

var (
	errNoShare     = errors.New("no such share")
	errShareExists = errors.New("the share exists")
	errShareEnded  = errors.New("the share ended")
	errRateLimited = errors.New("too many download URLs of this share")
)

func newShareEnded(version uint64) error {
	failure := connect.NewError(connect.CodePermissionDenied, errShareEnded)
	if detail, err := connect.NewErrorDetail(&syncv1.ShareEnded{Version: version}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}

func newRateLimited(retryAfter time.Duration) error {
	failure := connect.NewError(connect.CodeResourceExhausted, errRateLimited)
	if detail, err := connect.NewErrorDetail(&syncv1.RateLimited{RetryAfterSeconds: uint32(retryAfter.Seconds())}); err == nil {
		failure.AddDetail(detail)
	}
	return failure
}

func newNoShare() error { return connect.NewError(connect.CodeNotFound, errNoShare) }
