// Package savedsearch writes saved search notes: it creates them and changes their query for the agent
// tools and the tab.
package savedsearch

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/searchnote"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/textplace"
)

var errEmptyTitle = errors.New("title is empty")

// Runner runs a saved search at once, waiting for the run or in the background, and checks a query on the
// core.
type Runner interface {
	RunNow(ctx context.Context, vault domain.VaultID, notePath string) (resultcache.Record, error)
	StartRunNow(ctx context.Context, vault domain.VaultID, notePath string)
	CheckQuery(ctx context.Context, vault domain.VaultID, query string) ([]api.Problem, error)
}

// Settings are the folder a note goes to when the call names none.
type Settings struct {
	DefaultFolder string
}

// Dependencies are the notes of the vaults, the runner of the searches, the settings at the time of a call
// and the clock that stamps the id of a new note.
type Dependencies struct {
	Open     func(id domain.VaultID) ports.Notes
	Runner   Runner
	Settings func() Settings
	Clock    ports.Clock
}

// UseCase writes saved search notes.
type UseCase struct {
	open     func(id domain.VaultID) ports.Notes
	runner   Runner
	settings func() Settings
	clock    ports.Clock
}

// New returns the use case over the dependencies.
func New(dependencies Dependencies) *UseCase {
	return &UseCase{open: dependencies.Open, runner: dependencies.Runner, settings: dependencies.Settings, clock: dependencies.Clock}
}

// Create writes a new saved search note named after its title, in the folder of the input or of the
// setting, scans it, and runs it at once. A query the core reports problems of returns them in the output
// and writes nothing.
func (u *UseCase) Create(ctx context.Context, vault domain.VaultID, in api.CreateInput) (api.SavedOutput, error) {
	settings := u.settings()
	problems, err := u.runner.CheckQuery(ctx, vault, in.Query)
	if err != nil || len(problems) > 0 {
		return newProblemOutput(problems), err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return api.SavedOutput{}, errEmptyTitle
	}
	name, exact, err := domain.Filename(title)
	if err != nil {
		return api.SavedOutput{}, err
	}
	folder := in.Folder
	if strings.TrimSpace(folder) == "" {
		folder = settings.DefaultFolder
	}
	notePath := path.Join(strings.Trim(folder, "/"), name+domain.NoteExtension)
	id, err := ulid.NewGenerator(rand.Reader).NewID(u.clock.Now())
	if err != nil {
		return api.SavedOutput{}, err
	}
	definition := searchnote.Definition{ID: id, Title: title, Query: in.Query}
	raw, err := searchnote.Create(id, title, exact, definition)
	if err != nil {
		return api.SavedOutput{}, err
	}
	notes := u.open(vault)
	if err := notes.Create(ctx, notePath, raw); err != nil {
		if errors.Is(err, host.ErrAlreadyExists) {
			return api.SavedOutput{}, fmt.Errorf("a file is at %s; choose another title", notePath)
		}
		return api.SavedOutput{}, err
	}
	if err := notes.Scan(ctx, []string{notePath}); err != nil {
		return api.SavedOutput{}, err
	}
	return u.runFirst(ctx, vault, notePath)
}

func newProblemOutput(problems []api.Problem) api.SavedOutput {
	return api.SavedOutput{
		Path: "", URI: "", ResultCount: 0, Order: "", ComputedAt: "", Warnings: nil, Errors: problems,
	}
}

func (u *UseCase) runFirst(ctx context.Context, vault domain.VaultID, notePath string) (api.SavedOutput, error) {
	record, err := u.runner.RunNow(ctx, vault, notePath)
	if err != nil && ctx.Err() != nil {
		return api.SavedOutput{}, ctx.Err()
	}
	out := api.SavedOutput{
		Path: notePath, URI: textplace.FormatNoteAddress(notePath), ResultCount: len(record.Results), Order: record.Order,
		ComputedAt: "", Warnings: append([]string(nil), record.Warnings...), Errors: nil,
	}
	if !record.ComputedAt.IsZero() {
		out.ComputedAt = record.ComputedAt.UTC().Format(time.RFC3339)
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
