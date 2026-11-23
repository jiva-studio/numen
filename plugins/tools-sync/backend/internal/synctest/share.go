package synctest

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
)

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

func (c *Computer) GetReaderCode(ctx context.Context) (ReaderCode, error) {
	code, err := receive.NewGetReaderCode(c.ReceiveDeps()).Execute(ctx)
	return ReaderCode{Code: code.Code, Fingerprint: code.Fingerprint}, err
}

// CreateShare shares the folder of the vault and waits until its files are rewrapped.
func (c *Computer) CreateShare(ctx context.Context, id domain.VaultID, folder string) error {
	return share.NewCreateShare(c.ShareDeps()).Execute(ctx, id, folder)
}

// AddReader adds the reader of the code to the sharing of the folder and returns the invitation.
func (c *Computer) AddReader(ctx context.Context, id domain.VaultID, folder, code, label string, isConfirmed bool) (ShareInvitation, error) {
	invited, err := share.NewAddReader(c.ShareDeps()).Execute(ctx, share.AddReaderInput{
		VaultID: id, Folder: folder, Code: code, Label: label, IsConfirmed: isConfirmed,
	})
	return ShareInvitation{Text: invited.Text, ReaderFingerprint: invited.Fingerprint}, err
}

// RemoveReader removes the reader with the account fingerprint from the sharing of the folder.
func (c *Computer) RemoveReader(ctx context.Context, id domain.VaultID, folder, fingerprint string) error {
	return share.NewRemoveReader(c.ShareDeps()).Execute(ctx, id, folder, fingerprint)
}

func (c *Computer) StopShare(ctx context.Context, id domain.VaultID, folder string) error {
	return share.NewStopShare(c.ShareDeps()).Execute(ctx, id, folder)
}

// GetShare returns the sharing of the folder as the owner lists it.
func (c *Computer) GetShare(ctx context.Context, id domain.VaultID, folder string) (ShareState, error) {
	overview, err := share.NewGetShare(c.ShareDeps()).Execute(ctx, id, folder)
	state := ShareState{State: overview.State, Readers: nil}
	for _, reader := range overview.Readers {
		state.Readers = append(state.Readers, SharedReader{Label: reader.Label, Fingerprint: reader.Fingerprint, State: reader.GetState()})
	}
	return state, err
}

// ReadInvitation shows what an invitation offers and stores nothing.
func (c *Computer) ReadInvitation(ctx context.Context, text string) (SharePreview, error) {
	preview, err := receive.NewReadInvitation(c.ReceiveDeps()).Execute(ctx, text)
	return SharePreview{
		ShareName: preview.ShareName, OwnerFingerprint: preview.OwnerFingerprint, IsForThisAccount: preview.IsForThisAccount,
		AlreadyReceived: preview.IsReceived,
	}, err
}

// AcceptInvitation receives the share of the invitation as a vault managed by the sync plugin and runs its first pass.
func (c *Computer) AcceptInvitation(ctx context.Context, text string) (ReceivedShare, error) {
	accepted, err := receive.NewAcceptInvitation(c.ReceiveDeps()).Execute(ctx, text)
	return ReceivedShare{VaultID: accepted.VaultID, Name: accepted.Name, OwnerFingerprint: "", State: "", ConflictCount: 0}, err
}

func (c *Computer) ListReceived(ctx context.Context) ([]ReceivedShare, error) {
	listed, err := receive.NewListReceived(c.ReceiveDeps()).Execute(ctx)
	shares := make([]ReceivedShare, len(listed))
	for i, one := range listed {
		shares[i] = ReceivedShare{
			VaultID: one.VaultID, Name: one.Name, OwnerFingerprint: one.OwnerFingerprint, State: one.State, ConflictCount: one.ConflictCount,
		}
	}
	return shares, err
}

// ConnectVault connects the vault as a new synced vault through the use case of the plugin.
func (c *Computer) ConnectVault(ctx context.Context, id domain.VaultID) error {
	return catalog.NewConnectVault(c.CatalogDeps()).Execute(ctx, catalog.ConnectInput{VaultID: id, Mode: catalog.ModeNew, RemoteID: ""})
}

// MoveNote moves a file of the vault to a folder the way the host moves a note.
func (c *Computer) MoveNote(t *testing.T, id domain.VaultID, from, folder string) {
	t.Helper()
	if _, err := c.Vaults.With(id).MoveNotes(t.Context(), []string{from}, folder); err != nil {
		t.Fatal(err)
	}
}
