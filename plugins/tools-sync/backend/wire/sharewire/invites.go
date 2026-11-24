package sharewire

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
)

// FolderSharer is the sharing of folders over the use cases of sharing.
type FolderSharer struct{ deps share.Deps }

// NewFolderSharer returns the sharing over the dependencies.
func NewFolderSharer(deps share.Deps) *FolderSharer { return &FolderSharer{deps: deps} }

// CheckShareable refuses a folder that may not be shared.
func (s *FolderSharer) CheckShareable(ctx context.Context, id domain.VaultID, folder string) error {
	_, err := share.NewGetShare(s.deps).Execute(ctx, id, folder)
	return err
}

// ShareWithReader shares the folder when it is not shared yet, adds the reader of the code, and returns the text of the
// invitation.
func (s *FolderSharer) ShareWithReader(ctx context.Context, id domain.VaultID, folder, code, label string) (string, error) {
	invited, err := share.NewShareFolder(s.deps).Execute(ctx, share.AddReaderInput{
		VaultID: id, Folder: folder, Code: code, Label: label, IsConfirmed: false,
	})
	return invited.Text, err
}

// InvitationReceiver is the receiving of folders over the use cases of receiving.
type InvitationReceiver struct{ deps receive.Deps }

// NewInvitationReceiver returns the receiving over the dependencies.
func NewInvitationReceiver(deps receive.Deps) *InvitationReceiver {
	return &InvitationReceiver{deps: deps}
}

// GetInvitationOwner verifies the invitation, stores nothing, and returns the fingerprint of the account that shares.
func (r *InvitationReceiver) GetInvitationOwner(ctx context.Context, text string) (string, error) {
	preview, err := receive.NewReadInvitation(r.deps).Execute(ctx, text)
	return preview.OwnerFingerprint, err
}

// AcceptInvitation receives the folder the invitation names.
func (r *InvitationReceiver) AcceptInvitation(ctx context.Context, text string) error {
	_, err := receive.NewAcceptInvitation(r.deps).Execute(ctx, text)
	return err
}
