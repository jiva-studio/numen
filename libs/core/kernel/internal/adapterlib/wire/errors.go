package wire

import (
	"errors"
	"io/fs"
	"strings"

	"connectrpc.com/connect"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// ErrNoVault is a call that needs a vault while the window shows none.
var ErrNoVault = errors.New("the window has no vault")

// ErrTooManyPaths is a filter naming more paths than are answered at once.
var ErrTooManyPaths = errors.New("that is more paths than are answered at once")

const maxQueryLength = 200

// NewInternalError is the failure of a call the window could not serve for a reason of its own.
func NewInternalError(err error) error {
	return connect.NewError(connect.CodeInternal, err)
}

// GetDistinctPaths is the paths without the repeats, in the order they came, or the invalid argument of a filter
// naming more paths than are answered at once.
func GetDistinctPaths(paths []string) ([]string, error) {
	observed := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if observed[path] {
			continue
		}
		observed[path] = true
		out = append(out, path)
	}
	if len(out) > maxQueryLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrTooManyPaths)
	}
	return out, nil
}

// GetErrorDetail is the reason an error carries after the text of the first sentinel it wraps, and empty
// for an error that wraps none: the text of any other error is not for a client.
func GetErrorDetail(err error, sentinels ...error) string {
	for _, sentinel := range sentinels {
		if errors.Is(err, sentinel) {
			return strings.TrimPrefix(err.Error(), sentinel.Error()+": ")
		}
	}
	return ""
}

// NewListError is the failure of a call that reads a path of the vault: a path outside it is the caller's
// to correct, a path with nothing at it is not found.
func NewListError(err error) error {
	switch {
	case errors.Is(err, vaultdomain.ErrOutside):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, fs.ErrNotExist):
		return connect.NewError(connect.CodeNotFound, err)
	}
	return NewInternalError(err)
}
