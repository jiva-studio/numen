package fakeserver

import (
	"bytes"
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

type shareMember struct {
	Account string `json:"account"`
	Root    []byte `json:"root"`
	Added   uint64 `json:"added"`
	Removed uint64 `json:"removed"`
}

func (m *shareMember) isActive() bool { return m.Removed == 0 }

type shareState struct {
	account  string
	vault    string
	epoch    uint32
	isClosed bool
	log      []*syncv1.ShareLogEntry
	members  []*shareMember
	exits    map[string]uint64
	urls     map[int64]int
}

func newShareState(account, vault string) *shareState {
	return &shareState{account: account, vault: vault, exits: map[string]uint64{}, urls: map[int64]int{}}
}

func (sh *shareState) getHead() *syncv1.ShareLogEntry { return sh.log[len(sh.log)-1] }

func (sh *shareState) findMember(account string) *shareMember {
	for _, member := range sh.members {
		if member.Account == account {
			return member
		}
	}
	return nil
}

func (sh *shareState) findMemberByRoot(root []byte) *shareMember {
	for _, member := range sh.members {
		if bytes.Equal(member.Root, root) {
			return member
		}
	}
	return nil
}

func (sh *shareState) listActiveMembers() []*shareMember {
	var active []*shareMember
	for _, member := range sh.members {
		if member.isActive() {
			active = append(active, member)
		}
	}
	return active
}

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
