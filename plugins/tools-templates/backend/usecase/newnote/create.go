// Package newnote makes a note from a template.
package newnote

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	pathpkg "path"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/engine"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/mask"
)

const (
	noteExtension     = ".md"
	templateExtension = ".template"
	maxNames          = 100
)

var (
	// ErrNotATemplate is a path that is not a note any mask of the templates setting names.
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

// Request names a template, the folder the note is made in and the values of the fields the template asks.
type Request struct {
	Template string
	Folder   string
	Values   map[string]string
}

// Result names the note that was made.
type Result struct {
	Path string
}

// CreateFromTemplate makes a note from a note the masks of the templates setting name.
type CreateFromTemplate struct {
	vault    Vault
	clock    Clock
	ids      IDs
	getMasks func() mask.Set
	renderer engine.Renderer
}

// NewCreateFromTemplate returns the use case. getMasks is read at each call, so a changed setting applies at once.
func NewCreateFromTemplate(vault Vault, clock Clock, ids IDs, getMasks func() mask.Set) CreateFromTemplate {
	return CreateFromTemplate{vault: vault, clock: clock, ids: ids, getMasks: getMasks, renderer: engine.New(nil)}
}

// Execute makes the note.
func (u CreateFromTemplate) Execute(ctx context.Context, in Request) (Result, error) {
	text, err := readTemplate(ctx, u.vault, u.getMasks(), in.Template)
	if err != nil {
		return Result{}, err
	}
	now := u.clock.Now()
	id, err := u.ids.NewID(now)
	if err != nil {
		return Result{}, fmt.Errorf("make the id of the note: %w", err)
	}

	stem := getName(in.Template)
	for number := 1; number <= maxNames; number++ {
		name := stem
		if number > 1 {
			name = fmt.Sprintf("%s %d", stem, number)
		}
		path, isMade, err := u.createAs(ctx, pathpkg.Join(in.Folder, name+noteExtension), render{renderer: u.renderer, template: in.Template, text: text, data: engine.Data{Title: name, Folder: in.Folder, ID: id, Now: now, Values: in.Values}})
		if err != nil {
			return Result{}, err
		}
		if isMade {
			return Result{Path: path}, nil
		}
	}
	return Result{}, fmt.Errorf("%w %s", ErrNoFreeName, stem)
}

func (u CreateFromTemplate) createAs(ctx context.Context, path string, r render) (string, bool, error) {
	isFree, err := u.isFree(ctx, path)
	if err != nil || !isFree {
		return "", false, err
	}
	content, err := r.getContent()
	if err != nil {
		return "", false, err
	}
	err = u.vault.Create(ctx, path, content)
	if errors.Is(err, host.ErrAlreadyExists) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("create the note %s: %w", path, err)
	}
	if err := u.vault.Scan(ctx, []string{path}); err != nil {
		return "", false, fmt.Errorf("index the note %s: %w", path, err)
	}
	return path, true, nil
}

func (u CreateFromTemplate) isFree(ctx context.Context, path string) (bool, error) {
	_, err := u.vault.StatFile(ctx, path)
	if errors.Is(err, host.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("check the name %s: %w", path, err)
	}
	return false, nil
}

func isTemplate(path string, masks mask.Set) bool {
	if path != pathpkg.Clean(path) || !strings.HasSuffix(path, noteExtension) || strings.TrimSuffix(pathpkg.Base(path), noteExtension) == "" {
		return false
	}
	_, isNamed := masks.Find(path)
	return isNamed
}

func getName(template string) string {
	stem := strings.TrimSuffix(pathpkg.Base(template), noteExtension)
	if name := strings.TrimSuffix(stem, templateExtension); name != "" {
		return name
	}
	return stem
}

// render is what a note is made from: the template and its text, and what the template reads.
type render struct {
	renderer engine.Renderer
	template string
	text     string
	data     engine.Data
}

func (r render) getContent() ([]byte, error) {
	rendered, err := r.renderer.Render(pathpkg.Base(r.template), r.text, r.data)
	if err != nil {
		return nil, wrapRenderError(err)
	}
	doc, err := markdown.Open([]byte(rendered.Text))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidNote, err)
	}
	if err := doc.SetIdentifier(r.data.ID); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidNote, err)
	}
	return doc.Bytes(), nil
}

func wrapRenderError(err error) error {
	if errors.Is(err, engine.ErrTemplate) {
		return fmt.Errorf("%w: %w", ErrInvalidNote, err)
	}
	return err
}

func readTemplate(ctx context.Context, vault Vault, masks mask.Set, path string) (string, error) {
	if !isTemplate(path, masks) {
		return "", fmt.Errorf("%w: %s", ErrNotATemplate, path)
	}
	raw, _, err := vault.Read(ctx, path)
	if err != nil {
		return "", fmt.Errorf("read the template %s: %w", path, err)
	}
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("%w: it is not UTF-8 text", ErrInvalidNote)
	}
	return string(raw), nil
}

// NewIDs returns the generator of the ids of new notes, seeded by the system.
func NewIDs() IDs { return ulid.NewGenerator(rand.Reader) }

// Masks holds the masks of the templates setting, which the host changes while calls are being served.
type Masks struct {
	value atomic.Pointer[mask.Set]
}

// NewMasks returns masks that hold initial.
func NewMasks(initial mask.Set) *Masks {
	masks := new(Masks)
	masks.Set(initial)
	return masks
}

// Set replaces the masks.
func (m *Masks) Set(set mask.Set) { m.value.Store(&set) }

func (m *Masks) Get() mask.Set { return *m.value.Load() }
