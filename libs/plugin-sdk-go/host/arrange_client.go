package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// errNotImplemented is what the calls that rename, move and remove notes, run the checks of a vault
// and change the list of vaults return until the client makes them.
var errNotImplemented = fmt.Errorf("host: not implemented: %w", errors.ErrUnsupported)

var _ VaultList = (*Client)(nil)

func (c *Client) RenameNote(context.Context, string, string) (RenamedNote, error) {
	return RenamedNote{}, errNotImplemented
}

func (c *Client) MoveNotes(context.Context, []string, string) ([]NoteMove, error) {
	return nil, errNotImplemented
}

func (c *Client) RemoveNotes(context.Context, []string) ([]NoteRemoval, error) {
	return nil, errNotImplemented
}

func (c *Client) GetProblems(context.Context, []string) (VaultProblems, error) {
	return VaultProblems{}, errNotImplemented
}

func (c *Client) GetVaultList(context.Context) ([]KnownVault, error) {
	return nil, errNotImplemented
}

func (c *Client) AddVault(context.Context, string, string) (domain.Vault, error) {
	return domain.Vault{}, errNotImplemented
}

func (c *Client) RenameVault(context.Context, domain.VaultID, string) (domain.Vault, error) {
	return domain.Vault{}, errNotImplemented
}

func (c *Client) RemoveVault(context.Context, domain.VaultID) (string, error) {
	return "", errNotImplemented
}

func (c *Client) OpenVault(context.Context, domain.VaultID) error {
	return errNotImplemented
}
