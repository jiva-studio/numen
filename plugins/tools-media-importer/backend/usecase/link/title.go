package link

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
)

const (
	titleTimeout = 15 * time.Second
	maxSuffix    = 999
)

func (l *Links) getPageTitle(ctx context.Context, parsed address.URL) string {
	fetchCtx, cancel := l.withClosing(ctx)
	defer cancel()
	fetchCtx, cancelTimeout := context.WithTimeout(fetchCtx, titleTimeout)
	defer cancelTimeout()
	meta, err := l.fetcher().GetMetadata(fetchCtx, string(parsed))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Title)
}

func (l *Links) createTitledNote(ctx context.Context, vault domain.VaultID, folder string, parsed address.URL, title string) (Link, error) {
	base, isExact, err := domain.CreateFilename(title)
	if title == "" || err != nil {
		base, isExact, err = domain.CreateFilename(string(parsed))
		title = ""
	}
	if err != nil {
		return Link{}, err
	}
	if isExact {
		title = ""
	}
	for suffix := 0; suffix <= maxSuffix; suffix++ {
		name := base
		if suffix > 0 {
			name = fmt.Sprintf("%s %d", base, suffix)
		}
		made, err := l.createNoteAs(ctx, vault, folder, parsed, name, title)
		if !errors.Is(err, host.ErrAlreadyExists) {
			return made, err
		}
	}
	return Link{}, fmt.Errorf("no free file name for %s in %s", base, folder)
}
