// Package savedsearch writes saved search notes: it creates them and changes their query for the agent
// tools and the tab.
package savedsearch

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/domain/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/searchnote"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/textplace"
)

var errEmptyTitle = errors.New("title is empty")

// Runner runs a saved search at once, waiting for the run or in the background, and checks a query on the
// core.
type Runner interface {
	RunNow(ctx context.Context, vault domain.VaultID, notePath string) (resultcache.Record, error)
	StartRunNow(ctx context.Context, vault domain.VaultID, notePath string)
	CheckQuery(ctx context.Context, vault domain.VaultID, query string) ([]resultcache.Problem, error)
	CheckNoteQuery(ctx context.Context, vault domain.VaultID, notePath string, query string) ([]resultcache.Problem, error)
}

// CreateInput is the title, the folder and the query of a new saved search note.
type CreateInput struct {
	Title  string
	Folder string
	Query  string
}

// SavedSearch is what a write of a saved search note produced: the note, the address of the note, the results of
// its first run and their order, the time of the run, zero when there was none, and the warnings of the
// run. Errors are the problems of a query that is not valid; nothing is written when there are any.
type SavedSearch struct {
	Path        string
	URI         string
	ResultCount int
	Order       resultcache.Order
	ComputedAt  time.Time
	Warnings    []string
	Errors      []resultcache.Problem
}

// Settings are the folder a note goes to when the call names none.
type Settings struct {
	DefaultFolder string
}

// Dependencies are the notes of the vaults, the runner of the searches, the settings at the time of a call,
// the clock that stamps the id of a new note and the entropy of its random bits.
type Dependencies struct {
	Open        func(id domain.VaultID) ports.Notes
	Runner      Runner
	GetSettings func() Settings
	Clock       ports.Clock
	Entropy     ports.Entropy
}

// UseCase writes saved search notes.
type UseCase struct {
	open        func(id domain.VaultID) ports.Notes
	runner      Runner
	getSettings func() Settings
	clock       ports.Clock
	entropy     ports.Entropy
}

// New returns the use case over the dependencies.
func New(dependencies Dependencies) *UseCase {
	return &UseCase{
		open: dependencies.Open, runner: dependencies.Runner, getSettings: dependencies.GetSettings, clock: dependencies.Clock,
		entropy: dependencies.Entropy,
	}
}

// Create writes a new saved search note named after its title, in the folder of the input or of the
// setting, scans it, and runs it at once; a note with no query is written and not run. A query the core reports problems of returns them in the output
// and writes nothing.
func (u *UseCase) Create(ctx context.Context, vault domain.VaultID, in CreateInput) (SavedSearch, error) {
	settings := u.getSettings()
	hasQuery := strings.TrimSpace(in.Query) != ""
	if problems, err := u.checkQuery(ctx, vault, in.Query); err != nil || len(problems) > 0 {
		return newProblemOutput(problems), err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return SavedSearch{}, errEmptyTitle
	}
	name, exact, err := domain.CreateFilename(title)
	if err != nil {
		return SavedSearch{}, err
	}
	folder := in.Folder
	if strings.TrimSpace(folder) == "" {
		folder = settings.DefaultFolder
	}
	notePath := path.Join(strings.Trim(folder, "/"), name+domain.NoteExtension)
	id, err := ulid.NewGenerator(entropyReader{fill: u.entropy.Fill}).NewID(u.clock.Now())
	if err != nil {
		return SavedSearch{}, err
	}
	definition := searchnote.Definition{ID: id, Title: title, Query: in.Query}
	raw, err := searchnote.Create(id, title, exact, definition)
	if err != nil {
		return SavedSearch{}, err
	}
	notes := u.open(vault)
	if err := notes.Create(ctx, notePath, raw); err != nil {
		if errors.Is(err, host.ErrAlreadyExists) {
			return SavedSearch{}, fmt.Errorf("a file is at %s; choose another title", notePath)
		}
		return SavedSearch{}, err
	}
	if err := notes.Scan(ctx, []string{notePath}); err != nil {
		return SavedSearch{}, err
	}
	if !hasQuery {
		return SavedSearch{
			Path: notePath, URI: textplace.FormatNoteAddress(notePath), ResultCount: 0, Order: "", ComputedAt: time.Time{},
			Warnings: nil, Errors: nil,
		}, nil
	}
	return u.runFirst(ctx, vault, notePath)
}

func (u *UseCase) checkQuery(ctx context.Context, vault domain.VaultID, query string) ([]resultcache.Problem, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	return u.runner.CheckQuery(ctx, vault, query)
}

func newProblemOutput(problems []resultcache.Problem) SavedSearch {
	return SavedSearch{
		Path: "", URI: "", ResultCount: 0, Order: "", ComputedAt: time.Time{}, Warnings: nil, Errors: problems,
	}
}

func (u *UseCase) runFirst(ctx context.Context, vault domain.VaultID, notePath string) (SavedSearch, error) {
	record, err := u.runner.RunNow(ctx, vault, notePath)
	if err != nil && ctx.Err() != nil {
		return SavedSearch{}, ctx.Err()
	}
	out := SavedSearch{
		Path: notePath, URI: textplace.FormatNoteAddress(notePath), ResultCount: len(record.Results), Order: record.Order,
		ComputedAt: record.ComputedAt, Warnings: append([]string(nil), record.Warnings...), Errors: nil,
	}
	for _, problem := range record.Errors {
		out.Warnings = append(out.Warnings, fmt.Sprintf("%d:%d %s", problem.Line, problem.Column, problem.Message))
	}
	switch {
	case record.Failure != "":
		out.Warnings = append(out.Warnings, "the run failed: "+record.Failure)
	case err != nil:
		out.Warnings = append(out.Warnings, "the run failed: "+err.Error())
	}
	return out, nil
}

type entropyReader struct{ fill func(b []byte) }

func (r entropyReader) Read(b []byte) (int, error) {
	r.fill(b)
	return len(b), nil
}
