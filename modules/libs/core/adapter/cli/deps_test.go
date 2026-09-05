package cli_test

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/pdf"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
)

// derived is the shelf this installation keeps its own files on inside a vault:
// what a reading wrote, and what a transcription wrote.
func derived(options filesystem.Options) filesystem.DerivedStores {
	return filesystem.DerivedStores{
		Options: options,
		Area:    filesystem.OCRDir,
		Areas:   []string{filesystem.SpeechDir},
	}
}

// deps is what the commands of one session run through. The test assembles it
// itself: a registry and an index of its own, the folders it made, and a trash
// nothing of this machine's is sent to.
func (s *session) deps(where cli.Locations) cli.Deps {
	options := filesystem.Options{ServiceDir: where.ServiceDir}
	return cli.Deps{
		Vaults: func() (cli.Vaults, error) {
			return cli.Vaults{
				Registry: appstate.At(where.Registry),
				Readers:  filesystem.VaultReaders{Options: options},
				Identity: filesystem.VaultIdentity{Options: options},
				Trash:    s.bin,
				Now:      time.Now,
				Rows: func(ctx context.Context) (cli.VaultRows, error) {
					db, err := index.Open(ctx, where.Index)
					if err != nil {
						return cli.VaultRows{}, err
					}
					return cli.VaultRows{Vaults: db.Vaults(), Close: db.Close}, nil
				},
			}, nil
		},

		Links: func(ctx context.Context) (cli.Links, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Links{}, err
			}
			return cli.Links{Show: note.NewShowLinks(db.NoteQueries()), Close: db.Close}, nil
		},

		// No embedder: a test reaches no model, so a question is answered by its
		// words alone.
		Search: func(ctx context.Context, trouble port.Trouble) (cli.Search, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Search{}, err
			}
			return cli.Search{
				Search: search.New(db.ChunkQueries(), filesystem.VaultReaders{Options: options},
					derived(options), pdf.Documents{}, nil, 0, trouble),
				Close: db.Close,
			}, nil
		},

		Problems: func(ctx context.Context) (cli.Problems, error) {
			db, err := index.Open(ctx, where.Index)
			if err != nil {
				return cli.Problems{}, err
			}
			return cli.Problems{Problems: db.NoteQueries(), Close: db.Close}, nil
		},
	}
}
