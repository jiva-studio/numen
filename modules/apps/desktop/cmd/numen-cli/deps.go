package main

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// errNoTools is a machine holding neither of the programs an address is
// downloaded with. The settings name where each of them is.
var errNoTools = errors.New("neither yt-dlp nor ffmpeg is on this machine")

// makeDeps is what the terminal works through, built for the places one run was
// pointed at. An installation is assembled by the application, so the openers
// are made here and the terminal is handed them.
//
// Each one opens when its command asks and not before: a person listing vaults
// waits for no index, and one asking about a scan waits for no transcriber.
func makeDeps(cfg container.Config) func(cli.Locations) cli.Deps {
	return func(where cli.Locations) cli.Deps {
		cfg.IndexPath = where.Index
		cfg.RegistryPath = where.Registry
		cfg.ServiceDir = where.ServiceDir
		if chosen, err := cfg.Settings(); err == nil {
			cfg = cfg.SetSettings(chosen)
		}

		return cli.Deps{
			Vaults: func() (cli.Vaults, error) { return openVaults(cfg) },
			Links:  func(ctx context.Context) (cli.Links, error) { return openLinks(ctx, cfg) },
			Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
				return openScan(ctx, cfg, v, rebuild)
			},
			Recognise: func(ctx context.Context, v domain.Vault, fetching func()) (cli.Recognise, error) {
				return openRecognise(ctx, cfg, v, fetching)
			},
			Transcribe: func(ctx context.Context, v domain.Vault, fetching func()) (cli.Transcribe, error) {
				return openTranscribe(ctx, cfg, v, fetching)
			},
			ImportURL: func(ctx context.Context, _ domain.Vault) (cli.ImportURL, error) {
				return openImportURL(ctx, cfg)
			},
			ProofreadRecognition: func(ctx context.Context, v domain.Vault) (cli.ProofreadRecognition, error) {
				return openProofreadRecognition(ctx, cfg, v)
			},
			ProofreadTranscript: func(ctx context.Context, v domain.Vault) (cli.ProofreadTranscript, error) {
				return openProofreadTranscript(ctx, cfg, v)
			},
			Search: func(ctx context.Context, errorHandler port.ErrorHandler) (cli.Search, error) {
				return openSearch(ctx, cfg, errorHandler)
			},
			Problems: func(ctx context.Context) (cli.Problems, error) { return openProblems(ctx, cfg) },
		}
	}
}

// openVaults is the registry of vaults with the readers over them. The index
// behind the rows is opened only when the rows are asked for.
func openVaults(cfg container.Config) (cli.Vaults, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return cli.Vaults{}, err
	}
	return cli.Vaults{
		Registry: registry,
		Readers:  cfg.VaultReaders(),
		Identity: cfg.VaultIdentity(),
		Trash:    cfg.Trash(),
		Now:      cfg.Clock(),
		Rows: func(ctx context.Context) (cli.VaultRows, error) {
			db, err := cfg.OpenIndex(ctx)
			if err != nil {
				return cli.VaultRows{}, err
			}
			return cli.VaultRows{Vaults: db.Vaults(), Close: db.Close}, nil
		},
	}, nil
}

// openLinks is the links of a note, read from the index.
func openLinks(ctx context.Context, cfg container.Config) (cli.Links, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Links{}, err
	}
	notes := cfg.Notes(
		db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), cfg.Level(db))
	return cli.Links{Show: notes.Links, Close: db.Close}, nil
}

// openScan is the walk of a vault into the index. A terminal waits for the
// model, and a vault is read whether or not one arrives.
func openScan(
	ctx context.Context, cfg container.Config, v domain.Vault, rebuild bool,
) (cli.Scan, error) {
	indexing := cfg
	indexing.ShouldRebuildIndex = rebuild
	db, err := indexing.OpenIndex(ctx)
	if err != nil {
		return cli.Scan{}, err
	}
	embedder, closeEmbedder, why := indexing.Embedder(ctx)
	read, err := indexing.ReadWholeVault(ctx, db, embedder, v)
	if err != nil {
		closeIfOpen(closeEmbedder)
		_ = db.Close()
		return cli.Scan{}, err
	}
	return cli.Scan{
		Read: read,
		Summary: func(ctx context.Context) (domain.VaultSummary, error) {
			return db.Queries().GetSummary(ctx, v.ID)
		},
		Unembedded: why,
		Close: func() error {
			closeIfOpen(closeEmbedder)
			return db.Close()
		},
	}, nil
}

// closeIfOpen closes what a function returned to close, when it returned one.
func closeIfOpen(closeFn func() error) {
	if closeFn != nil {
		_ = closeFn()
	}
}

// openRecognise is the reading of scans with the models on this machine. The
// person is told through fetching when the models have to be downloaded first.
func openRecognise(
	ctx context.Context, cfg container.Config, v domain.Vault, fetching func(),
) (cli.Recognise, error) {
	//nolint:contextcheck // whether a model is on this machine is asked of the disk, and waits for nothing
	if !cfg.RecogniserReady() {
		fetching()
	}
	models, closeModels, why := cfg.Recogniser(ctx)
	if why != nil {
		return cli.Recognise{}, why
	}
	db, cut, err := openCut(ctx, cfg, v)
	if err != nil {
		_ = closeModels()
		return cli.Recognise{}, err
	}
	return cli.Recognise{
		Recognise: cfg.Recognise(db.Sources(), models),
		Cut:       cut,
		Close: func() error {
			_ = closeModels()
			return db.Close()
		},
	}, nil
}

// openTranscribe is the transcription of recordings with the models on this
// machine. The person is told through fetching when the models have to be
// downloaded first.
func openTranscribe(
	ctx context.Context, cfg container.Config, v domain.Vault, fetching func(),
) (cli.Transcribe, error) {
	//nolint:contextcheck // whether a model is on this machine is asked of the disk, and waits for nothing
	if !cfg.TranscriberReady() {
		fetching()
	}
	models, closeModels, why := cfg.Transcriber(ctx)
	if why != nil {
		return cli.Transcribe{}, why
	}
	db, cut, err := openCut(ctx, cfg, v)
	if err != nil {
		_ = closeModels()
		return cli.Transcribe{}, err
	}
	return cli.Transcribe{
		Transcribe: cfg.Transcribe(db.Sources(), models),
		Cut:        cut,
		Close: func() error {
			_ = closeModels()
			return db.Close()
		},
	}, nil
}

// openCut is the index with the extraction of a vault's sources over it. The
// index is closed when the extraction cannot be made.
func openCut(
	ctx context.Context, cfg container.Config, v domain.Vault,
) (*container.Index, source.Extract, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return nil, source.Extract{}, err
	}
	cut, err := cfg.Extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		_ = db.Close()
		return nil, source.Extract{}, err
	}
	return db, cut, nil
}

// openImportURL is the download of an address into the vault.
func openImportURL(ctx context.Context, cfg container.Config) (cli.ImportURL, error) {
	by := cfg.Downloader(ctx)
	if by == nil {
		return cli.ImportURL{}, errNoTools
	}
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.ImportURL{}, err
	}
	return cli.ImportURL{ImportURL: cfg.ImportURL(ctx, db, by), Close: db.Close}, nil
}

// openProofreadRecognition is the proofreading of recognised scans, when it is
// configured.
func openProofreadRecognition(
	ctx context.Context, cfg container.Config, v domain.Vault,
) (cli.ProofreadRecognition, error) {
	proofread, held, err := cfg.GetScanProofreading().
		Recognition(cfg.VaultReaders(), cfg.GetDerivedStores())
	if err != nil || !held {
		return cli.ProofreadRecognition{}, err
	}
	db, cut, err := openCut(ctx, cfg, v)
	if err != nil {
		return cli.ProofreadRecognition{}, err
	}
	return cli.ProofreadRecognition{
		Proofread: proofread, Cut: cut, IsHeld: true, Close: db.Close,
	}, nil
}

// openProofreadTranscript is the proofreading of transcripts, when it is
// configured.
func openProofreadTranscript(
	ctx context.Context, cfg container.Config, v domain.Vault,
) (cli.ProofreadTranscript, error) {
	proofread, held, err := cfg.GetSpeechProofreading().
		Transcript(cfg.VaultReaders(), cfg.GetDerivedStores())
	if err != nil || !held {
		return cli.ProofreadTranscript{}, err
	}
	db, cut, err := openCut(ctx, cfg, v)
	if err != nil {
		return cli.ProofreadTranscript{}, err
	}
	return cli.ProofreadTranscript{
		Proofread: proofread, Cut: cut, IsHeld: true, Close: db.Close,
	}, nil
}

// openSearch is the search over the index. Only the provider that embeds
// questions is opened, because nothing a search does fills an index.
func openSearch(
	ctx context.Context, cfg container.Config, errorHandler port.ErrorHandler,
) (cli.Search, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Search{}, err
	}
	asking, closeAsking, why := cfg.OpenQuestionEmbedder(ctx)
	return cli.Search{
		Search: cfg.NewSearch(db, asking, errorHandler),
		Words:  why,
		Close: func() error {
			closeIfOpen(closeAsking)
			return db.Close()
		},
	}, nil
}

// openProblems is the problems the index holds.
func openProblems(ctx context.Context, cfg container.Config) (cli.Problems, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	return cli.Problems{Problems: db.Problems(), Close: db.Close}, nil
}
