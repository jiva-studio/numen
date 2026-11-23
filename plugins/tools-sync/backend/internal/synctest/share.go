package synctest

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var errNotImplemented = errors.New("not implemented")

// ReaderCode is what a person gives the owner of a share: the code and the fingerprint shown beside it.
type ReaderCode struct {
	Code        string
	Fingerprint string
}

// ShareInvitation is what the owner sends: the text and the fingerprint of the reader it is for.
type ShareInvitation struct {
	Text              string
	ReaderFingerprint string
}

// SharePreview is what a reader sees of an invitation before it accepts.
type SharePreview struct {
	ShareName        string
	OwnerFingerprint string
	IsForThisAccount bool
	AlreadyReceived  bool
}

// ReceivedShare is a share this computer received: its vault, its name, the owner and how its last pass ended.
type ReceivedShare struct {
	VaultID          domain.VaultID
	Name             string
	OwnerFingerprint string
	State            string
	ConflictCount    int
}

// SharedReader is a reader of a shared folder as the owner lists it.
type SharedReader struct {
	Label       string
	Fingerprint string
	State       string
}

// ShareState is the sharing of one folder as the owner sees it.
type ShareState struct {
	State   string
	Readers []SharedReader
}

// NewReaderComputer makes a computer with a new identity of its own, signed in to the server, with a vault of the
// name. Its account holds no subscription.
func NewReaderComputer(t *testing.T, server *Server, clock *Clock, vault string) *Computer {
	t.Helper()
	return NewFirstComputer(t, server, clock, vault, NewWords(t))
}

// NewUnsignedComputer makes a computer with a new identity that has never signed in to the server.
func NewUnsignedComputer(t *testing.T, server *Server, clock *Clock, vault string) *Computer {
	t.Helper()
	c := newComputer(t, server, clock, vault)
	c.Fake.StartRestore(NewWords(t))
	if err := c.Client.CompleteRestore(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	return c
}

// GetReaderCode returns the reader code of this computer.
func (c *Computer) GetReaderCode(_ context.Context) (ReaderCode, error) {
	return ReaderCode{}, errNotImplemented
}

// CreateShare shares the folder of the vault and waits until its files are rewrapped.
func (c *Computer) CreateShare(_ context.Context, _ domain.VaultID, _ string) error {
	return errNotImplemented
}

// AddReader adds the reader of the code to the sharing of the folder and returns the invitation.
func (c *Computer) AddReader(_ context.Context, _ domain.VaultID, _, _, _ string, _ bool) (ShareInvitation, error) {
	return ShareInvitation{}, errNotImplemented
}

// RemoveReader removes the reader with the account fingerprint from the sharing of the folder.
func (c *Computer) RemoveReader(_ context.Context, _ domain.VaultID, _, _ string) error {
	return errNotImplemented
}

// StopShare stops the sharing of the folder.
func (c *Computer) StopShare(_ context.Context, _ domain.VaultID, _ string) error {
	return errNotImplemented
}

// GetShare returns the sharing of the folder as the owner lists it.
func (c *Computer) GetShare(_ context.Context, _ domain.VaultID, _ string) (ShareState, error) {
	return ShareState{}, errNotImplemented
}

// ReadInvitation shows what an invitation offers and stores nothing.
func (c *Computer) ReadInvitation(_ context.Context, _ string) (SharePreview, error) {
	return SharePreview{}, errNotImplemented
}

// AcceptInvitation receives the share of the invitation as a vault managed by the sync plugin and runs its first pass.
func (c *Computer) AcceptInvitation(_ context.Context, _ string) (ReceivedShare, error) {
	return ReceivedShare{}, errNotImplemented
}

// ListReceived lists the shares this computer received.
func (c *Computer) ListReceived(_ context.Context) ([]ReceivedShare, error) {
	return nil, errNotImplemented
}

// ConnectVault connects the vault as a new synced vault through the use case of the plugin.
func (c *Computer) ConnectVault(_ context.Context, _ domain.VaultID) error {
	return errNotImplemented
}

// MoveNote moves a file of the vault the way the host reports a move.
func (c *Computer) MoveNote(_ domain.VaultID, _, _ string) {}
