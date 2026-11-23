// Package newnote makes a note from a template.
package newnote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	pathpkg "path"
	"slices"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/engine"
	"github.com/jiva-studio/numen/plugins/tools-templates/backend/domain/mask"
)

const (
	noteExtension     = ".md"
	templateExtension = ".template"
	maxNames          = 100
	maxTemplateSize   = 256 << 10
)

var (
	// ErrNotATemplate is a path that is not a note any mask of the templates setting names.
	ErrNotATemplate = errors.New("not a template")

	// ErrInvalidNote is a template that does not produce a valid note.
	ErrInvalidNote = errors.New("template does not produce a valid note")

	// ErrNameTaken is a name a note of the folder already has. Nothing is written.
	ErrNameTaken = errors.New("a note of that name exists")

	// ErrInvalidFolder is a folder that is not a clean path inside the vault.
	ErrInvalidFolder = errors.New("not a folder of the vault")

	// ErrDifferentQuestions is an answer to a question the template does not ask for the name given, which is not the one its fields were listed with.
	ErrDifferentQuestions = errors.New("this template asks different questions for this name")

	// ErrTemplateChanged is a template that is not the text its fields were listed from.
	ErrTemplateChanged = errors.New("template changed, open the picker again")

	// ErrNotIndexed is a note that was made but did not reach the index; Result still names it.
	ErrNotIndexed = errors.New("note is made but not indexed")
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

// Request names a template, the folder the note is made in, the name the note is given and the values of the
// fields the template asks. Etag is the one ListFields gave for the template.
type Request struct {
	Template string
	Folder   string
	Name     string
	Values   map[string]string
	Etag     string
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
	if !isFolder(in.Folder) {
		return Result{}, fmt.Errorf("%w: %q", ErrInvalidFolder, in.Folder)
	}
	text, err := readTemplate(ctx, u.vault, u.getMasks(), in.Template)
	if err != nil {
		return Result{}, err
	}
	if ComputeEtag(text) != in.Etag {
		return Result{}, ErrTemplateChanged
	}
	name := strings.TrimSpace(in.Name)
	if err := checkName(name); err != nil {
		return Result{}, err
	}
	now := u.clock.Now()
	id, err := u.ids.NewID(now)
	if err != nil {
		return Result{}, fmt.Errorf("make the id of the note: %w", err)
	}
	data := engine.Data{Title: name, Folder: in.Folder, ID: id, Now: now, Values: in.Values}
	if err := u.checkQuestions(ctx, in.Template, text, data); err != nil {
		return Result{}, err
	}
	path, err := u.create(ctx, getNotePath(in.Folder, name), render{renderer: u.renderer, template: in.Template, text: text, data: data})
	if path == "" {
		return Result{}, err
	}
	return Result{Path: path}, err
}

func (u CreateFromTemplate) checkQuestions(ctx context.Context, template, text string, data engine.Data) error {
	fields, err := u.renderer.ListFields(ctx, pathpkg.Base(template), text, engine.Data{Title: data.Title, Folder: data.Folder, ID: data.ID, Now: data.Now, Values: nil})
	if err != nil {
		return wrapRenderError(err)
	}
	for label := range data.Values {
		if !slices.ContainsFunc(fields, func(field engine.Field) bool { return field.Label == label }) {
			return fmt.Errorf("%w: %q", ErrDifferentQuestions, label)
		}
	}
	return nil
}

func (u CreateFromTemplate) create(ctx context.Context, path string, r render) (string, error) {
	isNameFree, err := isFree(ctx, u.vault, path)
	if err != nil {
		return "", err
	}
	if !isNameFree {
		return "", fmt.Errorf("%w: %s", ErrNameTaken, path)
	}
	content, err := r.getContent(ctx)
	if err != nil {
		return "", err
	}
	err = u.vault.Create(ctx, path, content)
	if errors.Is(err, host.ErrAlreadyExists) {
		return "", fmt.Errorf("%w: %s", ErrNameTaken, path)
	}
	if err != nil {
		return "", fmt.Errorf("create the note %s: %w", path, err)
	}
	if err := u.vault.Scan(ctx, []string{path}); err != nil {
		return path, fmt.Errorf("%w: %s: %w", ErrNotIndexed, path, err)
	}
	return path, nil
}

func isTemplate(path string, masks mask.Set) bool {
	if path == "." || path != pathpkg.Clean(path) || !strings.HasSuffix(path, noteExtension) || strings.TrimSuffix(pathpkg.Base(path), noteExtension) == "" {
		return false
	}
	_, isNamed := masks.Find(path)
	return isNamed
}

// ComputeEtag is what tells a text of a template from another.
func ComputeEtag(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func isFolder(path string) bool {
	if path == "" {
		return true
	}
	if path == "." || path != pathpkg.Clean(path) || strings.ContainsAny(path, "\\\x00") || pathpkg.IsAbs(path) {
		return false
	}
	if strings.ContainsFunc(path, func(r rune) bool { return unicode.IsControl(r) || r == '\u2028' || r == '\u2029' }) {
		return false
	}
	return !slices.Contains(strings.Split(path, "/"), "..")
}

// render is what a note is made from: the template and its text, and what the template reads.
type render struct {
	renderer engine.Renderer
	template string
	text     string
	data     engine.Data
}

func (r render) getContent(ctx context.Context) ([]byte, error) {
	rendered, err := r.renderer.Render(ctx, pathpkg.Base(r.template), r.text, r.data)
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
	if len(raw) > maxTemplateSize {
		return "", fmt.Errorf("%w: the template is larger than %d KiB", ErrInvalidNote, maxTemplateSize>>10)
	}
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("%w: it is not UTF-8 text", ErrInvalidNote)
	}
	return string(raw), nil
}

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
