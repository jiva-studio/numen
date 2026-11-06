package hostvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// errNotImplemented is what the calls that rename, move and remove notes and run the checks of a
// vault return until they are served.
var errNotImplemented = fmt.Errorf("hostvault: not implemented: %w", errors.ErrUnsupported)

func (a Vault) RenameNote(context.Context, string, string) (string, bool, *domain.MoveResult, error) {
	return "", false, nil, errNotImplemented
}

func (a Vault) MoveNote(context.Context, string, string) (domain.MoveResult, error) {
	return domain.MoveResult{}, errNotImplemented
}

func (a Vault) RemoveNote(context.Context, string) (string, []string, error) {
	return "", nil, errNotImplemented
}

func (a Vault) GetProblems(context.Context, []domain.Check) ([]domain.Check, []domain.VaultProblem, error) {
	return nil, nil, errNotImplemented
}
