// Package sharestate is the state of one share: its signed log, its members and the download URLs it issued.
package sharestate

import (
	"bytes"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// Member is an account that reads the share, with the versions of the log that added and removed it.
type Member struct {
	Account string `json:"account"`
	Root    []byte `json:"root"`
	Added   uint64 `json:"added"`
	Removed uint64 `json:"removed"`
}

// IsActive reports whether the member was not removed.
func (m *Member) IsActive() bool { return m.Removed == 0 }

// State is the log of one share and what the server derives from it.
type State struct {
	Account  string
	Vault    string
	Epoch    uint32
	IsClosed bool
	Log      []*syncv1.ShareLogEntry
	Members  []*Member
	Exits    map[string]uint64
	URLs     map[int64]int
}

// New returns the state of a share of the vault that the account owns.
func New(account, vault string) *State {
	return &State{Account: account, Vault: vault, Exits: map[string]uint64{}, URLs: map[int64]int{}}
}

// GetHead returns the last entry of the log.
func (sh *State) GetHead() *syncv1.ShareLogEntry { return sh.Log[len(sh.Log)-1] }

// FindMember returns the member of the account, or nil.
func (sh *State) FindMember(account string) *Member {
	for _, member := range sh.Members {
		if member.Account == account {
			return member
		}
	}
	return nil
}

// FindMemberByRoot returns the member with the root, or nil.
func (sh *State) FindMemberByRoot(root []byte) *Member {
	for _, member := range sh.Members {
		if bytes.Equal(member.Root, root) {
			return member
		}
	}
	return nil
}

// ListActiveMembers returns the members that were not removed.
func (sh *State) ListActiveMembers() []*Member {
	var active []*Member
	for _, member := range sh.Members {
		if member.IsActive() {
			active = append(active, member)
		}
	}
	return active
}
