// Package plugin holds the values, constants and sentinel errors of the plugin host.
package plugin

import (
	"errors"
	"fmt"
	"io/fs"
)

var (
	// ErrNoKey means the store holds no key under the name.
	ErrNoKey = errors.New("no key is stored under that name")

	// ErrNoKeyStore means this machine has no store to keep a key in. A key is
	// never kept anywhere else, and the caller is notified.
	ErrNoKeyStore = errors.New("this machine has no key store")

	// ErrPathTraversal is returned when a path escapes the active vault boundary.
	ErrPathTraversal = errors.New("path traversal outside active vault is prohibited")

	// ErrPermissionDenied is returned when a plugin lacks the required permission.
	ErrPermissionDenied = errors.New("plugin permission denied")

	// ErrNoProvisioner is a requirement that no provisioner can make ready.
	ErrNoProvisioner = errors.New("no provisioner for the dependency")

	// ErrNotText is returned by a ranged read of bytes that are not UTF-8.
	ErrNotText = errors.New("the file is not text")

	// ErrUnsavedChanges is returned by OpenVault while the window prompts the user about unsaved
	// changes. The window keeps its vault.
	ErrUnsavedChanges = errors.New("the window has unsaved changes and is prompting the user; retry after the prompt closes")

	// ErrNotAttempted is the error of the links of a note that were not written because the call was
	// canceled first.
	ErrNotAttempted = errors.New("not attempted")

	// ErrNoVault is returned by a vault call while the host shows no vault.
	ErrNoVault = errors.New("the host is showing no vault")

	// ErrNoConnections is returned when the host was built without the operations on connections.
	ErrNoConnections = errors.New("the host has no operations on connections")

	// ErrNoWindow is returned by a window call when the host shows no window.
	ErrNoWindow = errors.New("the host has no window")
	// ErrInvalidFocus is returned by a focus call for an argument the use case rejects: an empty path,
	// a passage that starts before the text or is empty, or too many passages. The error wraps it
	// with the use case's message.
	ErrInvalidFocus = errors.New("invalid focus target")
	// ErrTabNotOpen is returned by ActivateTab for an id that no open tab has. It is also fs.ErrNotExist.
	ErrTabNotOpen = fmt.Errorf("no open tab has that id: %w", fs.ErrNotExist)
	// ErrTabNotShown is returned by ActivateTab when the window does not report the tab active
	// within the bound of the call. The error wraps it with the id of the tab.
	ErrTabNotShown = errors.New("the window did not show the tab in time")

	// ErrNoModel is a search call on an installation that has no embedding model.
	ErrNoModel = errors.New("the installation has no embedding model")
	// ErrRecipeMismatch is a search call that names a recipe other than the one of the model in use.
	ErrRecipeMismatch = errors.New("the recipe is not the one of the model in use")
	// ErrInvalidFloor is a floor outside the range above zero and up to one.
	ErrInvalidFloor = errors.New("a floor is a cosine similarity above 0 and at most 1")
	// ErrLimitExceeded is an embedding call with more texts, or longer ones, than one call embeds.
	ErrLimitExceeded = errors.New("the call exceeds the limits of one embedding call")
	// ErrEmbeddingFailed is an embedding model that could not answer.
	ErrEmbeddingFailed = errors.New("the embedding model could not answer")
	// ErrInvalidQuery is a query vector whose dimension count is not the one of the vectors in the index.
	ErrInvalidQuery = errors.New("a query vector does not match the vectors of the index")
)

// invalidFocusError says what the error of a use case says and is ErrInvalidFocus as well.
type invalidFocusError struct{ err error }

func (e invalidFocusError) Error() string { return e.err.Error() }

func (e invalidFocusError) Unwrap() []error { return []error{e.err, ErrInvalidFocus} }

// InvalidFocus is err as the rejection of a focus call: it keeps the message and the identity of err
// and is also ErrInvalidFocus.
func InvalidFocus(err error) error { return invalidFocusError{err: err} }
