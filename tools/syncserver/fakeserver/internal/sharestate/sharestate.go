// Package sharestate is the state of one share: its signed log, its members and the download URLs it issued.
package sharestate

import (
	"bytes"

	"github.com/jiva-studio/numen/libs/shared/identity"
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

func (sh *State) GetHead() *syncv1.ShareLogEntry { return sh.Log[len(sh.Log)-1] }

func (sh *State) FindMember(account string) *Member {
	for _, member := range sh.Members {
		if member.Account == account {
			return member
		}
	}
	return nil
}

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

// HasSealsFor reports whether the entry holds one seal for each active member except the skipped one, and no other.
func (sh *State) HasSealsFor(entry *syncv1.ShareLogEntry, skipped *Member) bool {
	wanted := map[string]bool{}
	for _, reader := range sh.ListActiveMembers() {
		if reader != skipped {
			wanted[string(reader.Root)] = true
		}
	}
	if len(entry.GetSeals()) != len(wanted) {
		return false
	}
	for _, seal := range entry.GetSeals() {
		if !wanted[string(seal.GetReaderRootSignKey())] {
			return false
		}
		delete(wanted, string(seal.GetReaderRootSignKey()))
	}
	return true
}

// GetRemoveReaderProblem returns why the entry is not a removal of a reader of the share, or an empty string.
func (sh *State) GetRemoveReaderProblem(entry *syncv1.ShareLogEntry) string {
	subject := sh.FindMemberByRoot(entry.GetSubjectRootSignKey())
	switch {
	case entry.GetEpoch() != sh.Epoch+1:
		return "a removal is the next epoch"
	case subject == nil || !subject.IsActive():
		return "the subject is not a reader"
	case !sh.HasSealsFor(entry, subject):
		return "a removal has a seal for each remaining reader"
	}
	return ""
}

// Apply appends the entry to the log and changes the epoch, the members and the closing as the entry says.
func (sh *State) Apply(entry *syncv1.ShareLogEntry) {
	version := entry.GetVersion()
	sh.Log = append(sh.Log, entry)
	sh.Epoch = entry.GetEpoch()
	switch entry.GetKind() {
	case syncv1.ShareLogKind_SHARE_LOG_KIND_ADD_READER:
		root := entry.GetSubjectRootSignKey()
		account := identity.RecoveryPublicKey{Sign: root}.Fingerprint().String()
		if member := sh.FindMemberByRoot(root); member != nil {
			member.Added, member.Removed = version, 0
			return
		}
		sh.Members = append(sh.Members, &Member{Account: account, Root: bytes.Clone(root), Added: version})
	case syncv1.ShareLogKind_SHARE_LOG_KIND_REMOVE_READER:
		sh.FindMemberByRoot(entry.GetSubjectRootSignKey()).Removed = version
	case syncv1.ShareLogKind_SHARE_LOG_KIND_CLOSE:
		for _, member := range sh.ListActiveMembers() {
			member.Removed = version
		}
		sh.IsClosed = true
	case syncv1.ShareLogKind_SHARE_LOG_KIND_CREATE, syncv1.ShareLogKind_SHARE_LOG_KIND_ROTATE,
		syncv1.ShareLogKind_SHARE_LOG_KIND_UPDATE, syncv1.ShareLogKind_SHARE_LOG_KIND_UNSPECIFIED:
	}
}
