// Package newnote makes a note from a template.
package newnote

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var (
	// ErrNotATemplate is a path that is not a note inside the templates folder.
	ErrNotATemplate = errors.New("not a template")

	// ErrInvalidNote is a template that does not produce a valid note.
	ErrInvalidNote = errors.New("template does not produce a valid note")

	// ErrNoFreeName is a template whose name is taken in the folder, with every number tried.
	ErrNoFreeName = errors.New("no free name for")
)

// Vault is the calls the use case makes on the vault.
type Vault interface {
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
	Scan(ctx context.Context, paths []string) error
}

// Clock gets the current time.
type Clock interface{ Now() time.Time }

// IDs makes the identifier of a note.
type IDs interface {
	NewID(at time.Time) (string, error)
}

// Request names a template and the folder the note is made in.
type Request struct {
	Template string
	Folder   string
}

// Result names the note that was made.
type Result struct {
	Path string
}

type CreateFromTemplate struct{}

// NewCreateFromTemplate returns the use case. getFolder is read at each call, so a changed setting applies at once.
func NewCreateFromTemplate(_ Vault, _ Clock, _ IDs, _ func() string) CreateFromTemplate {
	return CreateFromTemplate{}
}

// Execute makes the note.
func (CreateFromTemplate) Execute(_ context.Context, _ Request) (Result, error) {
	return Result{Path: ""}, errors.New("not implemented")
}
