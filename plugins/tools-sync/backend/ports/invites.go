package ports

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// FolderSharing is the sharing of folders of this account, which sharing by a link uses.
type FolderSharing interface {
	// CheckShareable refuses a folder that may not be shared.
	CheckShareable(ctx context.Context, id domain.VaultID, folder string) error
	// GetVaultName returns the name of the vault, which the link of the whole vault names as its folder.
	GetVaultName(ctx context.Context, id domain.VaultID) (string, error)
	// ShareWithReader shares the folder when it is not shared yet, adds the reader of the code, and returns the text of the
	// invitation.
	ShareWithReader(ctx context.Context, id domain.VaultID, folder, code, label string) (string, error)
}

// InvitationReceiving is the receiving of folders other people share with this account, which sharing by a link uses.
type InvitationReceiving interface {
	// GetInvitationOwner verifies the invitation, stores nothing, and returns the fingerprint of the account that shares.
	GetInvitationOwner(ctx context.Context, text string) (string, error)
	// AcceptInvitation receives the folder the invitation names.
	AcceptInvitation(ctx context.Context, text string) error
}
