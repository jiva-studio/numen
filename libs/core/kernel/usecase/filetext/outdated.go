package filetext

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
)

// outdated finds the sources whose text version is no longer the live one.
type outdated struct {
	sources sourceport.SourceRepository
	records sourceport.SourceRecords
}

func newOutdatedIndex(u Index) outdated { return outdated{sources: u.sources, records: u.queries.records} }

func (u outdated) dropOutdatedText(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	notePaths []string,
	res *vaultreading.IndexResult,
) ([]string, error) {
	cut, err := u.records.GetRecognizedSources(ctx, v.ID, domain.KindFile)
	if err != nil {
		return notePaths, fmt.Errorf("read index: %w", err)
	}
	for _, one := range cut {
		if err := ctx.Err(); err != nil {
			return notePaths, err
		}
		if err := u.recordIfOutdated(ctx, v, on, one, res); err != nil {
			return notePaths, err
		}
	}
	notes, err := u.records.GetRecognizedSources(ctx, v.ID, domain.KindNote)
	if err != nil {
		return notePaths, fmt.Errorf("read index: %w", err)
	}
	for _, one := range notes {
		if err := ctx.Err(); err != nil {
			return notePaths, err
		}
		outdated, err := u.isOutdated(ctx, on, one)
		if err != nil {
			return notePaths, err
		}
		if outdated {
			res.Outdated++
			notePaths = append(notePaths, one.Path)
		}
	}
	return notePaths, nil
}

func (u outdated) recordIfOutdated(
	ctx context.Context,
	v domain.Vault,
	on openVault,
	source sourcedomain.Text,
	res *vaultreading.IndexResult,
) error {
	ref, outdated, err := u.statIfOutdated(ctx, on, source)
	if err != nil || !outdated {
		return err
	}
	if err := u.sources.SaveSource(ctx, v.ID, domain.Source{FileStat: ref}); err != nil {
		return fmt.Errorf("record %s: %w", source.Path, err)
	}
	res.Outdated++
	return nil
}

func (u outdated) isOutdated(ctx context.Context, on openVault, source sourcedomain.Text) (bool, error) {
	_, outdated, err := u.statIfOutdated(ctx, on, source)
	return outdated, err
}

func (u outdated) statIfOutdated(
	ctx context.Context,
	on openVault,
	source sourcedomain.Text,
) (domain.FileStat, bool, error) {
	live, err := on.texts.FindLiveVersion(ctx, source.Path, source.ContentHash)
	switch {
	case err == nil && string(live) == source.Producer:
		return domain.FileStat{}, false, nil
	case err != nil && !errors.Is(err, text.ErrNoText):
		return domain.FileStat{}, false, fmt.Errorf("find the live version of %s: %w", source.Path, err)
	}
	ref, err := on.files.StatFile(ctx, source.Path)
	if edit.IsNoNote(err) {
		return domain.FileStat{}, false, nil
	}
	if err != nil {
		return domain.FileStat{}, false, fmt.Errorf("stat %s: %w", source.Path, err)
	}
	return ref, true, nil
}
