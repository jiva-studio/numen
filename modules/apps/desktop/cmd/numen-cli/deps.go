package main

import (
	"context"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func configureLocations(cfg container.Config, where cli.Locations) container.Config {
	cfg.IndexPath = where.Index
	cfg.RegistryPath = where.Registry
	cfg.ServiceDir = where.ServiceDir
	cfg = cfg.SetSafeMode(where.SafeMode)
	if selected, err := cfg.Settings(); err == nil {
		cfg = cfg.SetSettings(selected)
	}
	return cfg
}

// makeDeps is what the terminal works through, built for the places one run was
// pointed at. An installation is assembled by the application, so the openers
// are made here and the terminal is handed them.
//
// Each one opens when its command asks and not before: a person listing vaults
// waits for no index, and one asking about a scan waits for no transcriber.
func makeDeps(cfg container.Config) func(cli.Locations) cli.Deps {
	return func(where cli.Locations) cli.Deps {
		cfg = configureLocations(cfg, where)

		return cli.Deps{
			Vaults: func() (cli.Vaults, error) { return createVaults(cfg) },
			Links:  func(ctx context.Context) (cli.Links, error) { return createLinks(ctx, cfg) },
			Scan: func(ctx context.Context, v domain.Vault, rebuild bool) (cli.Scan, error) {
				cfg.SetPluginVaultPath(v.Path)
				return createScan(ctx, cfg, v, rebuild)
			},
			Recognise: func(ctx context.Context, v domain.Vault, fetching func()) (cli.Recognise, error) {
				cfg.SetPluginVaultPath(v.Path)
				return createRecognition(ctx, cfg, v, fetching)
			},
			Transcribe: func(ctx context.Context, v domain.Vault, fetching func()) (cli.Transcribe, error) {
				cfg.SetPluginVaultPath(v.Path)
				return createTranscription(ctx, cfg, v, fetching)
			},
			ImportURL: func(ctx context.Context, v domain.Vault) (cli.ImportURL, error) {
				cfg.SetPluginVaultPath(v.Path)
				return createImportURL(ctx, cfg)
			},
			ProofreadRecognition: func(ctx context.Context, v domain.Vault) (cli.ProofreadRecognition, error) {
				cfg.SetPluginVaultPath(v.Path)
				return createProofreadRecognition(ctx, cfg, v)
			},
			ProofreadTranscript: func(ctx context.Context, v domain.Vault) (cli.ProofreadTranscript, error) {
				cfg.SetPluginVaultPath(v.Path)
				return createProofreadTranscript(ctx, cfg, v)
			},
			Search: func(ctx context.Context, errorHandler port.ErrorHandler) (cli.Search, error) {
				return createSearch(ctx, cfg, errorHandler)
			},
			Problems: func(ctx context.Context) (cli.Problems, error) { return createProblems(ctx, cfg) },
		}
	}
}

// createVaults is the registry of vaults with the readers over them. The index
// behind the rows is opened only when the rows are asked for.
func createVaults(cfg container.Config) (cli.Vaults, error) {
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

// createLinks is the links of a note, read from the index.
func createLinks(ctx context.Context, cfg container.Config) (cli.Links, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Links{}, err
	}
	notes := cfg.Notes(
		db.Queries(), db.Links(), db.Sources(), db.SourcesKnown(), cfg.Level(db))
	return cli.Links{Show: notes.Links, Close: db.Close}, nil
}

// createScan is the walk of a vault into the index. The vault is read whether
// or not the embedding model arrives.
func createScan(
	ctx context.Context, cfg container.Config, v domain.Vault, rebuild bool,
) (cli.Scan, error) {
	cfg.SetPluginVaultPath(v.Path)
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

// createRecognition is the reading of scans with the models on this machine. The
func createRecognition(
	ctx context.Context, cfg container.Config, v domain.Vault, fetching func(),
) (cli.Recognise, error) {
	cfg.SetPluginVaultPath(v.Path)
	if !cfg.RecogniserReady() {
		fetching()
	}
	models, closeModels, why := cfg.Recogniser(ctx)
	if why != nil {
		return cli.Recognise{}, why
	}
	indexed, err := createCut(ctx, cfg, v, cfg.Extract)
	if err != nil {
		_ = closeModels()
		return cli.Recognise{}, err
	}
	db, cutFn := indexed.db, indexed.cut
	return cli.Recognise{
		Recognise: cfg.Recognise(db.Sources(), models),
		Cut:       cutFn,
		Close: func() error {
			_ = closeModels()
			return db.Close()
		},
	}, nil
}

// createTranscription is the transcription of recordings with the models on this
// downloaded first.
func createTranscription(
	ctx context.Context, cfg container.Config, v domain.Vault, fetching func(),
) (cli.Transcribe, error) {
	cfg.SetPluginVaultPath(v.Path)
	if !cfg.TranscriberReady() {
		fetching()
	}
	models, closeModels, why := cfg.Transcriber(ctx)
	if why != nil {
		return cli.Transcribe{}, why
	}
	indexed, err := createCut(ctx, cfg, v, cfg.Extract)
	if err != nil {
		_ = closeModels()
		return cli.Transcribe{}, err
	}
	db, cutFn := indexed.db, indexed.cut
	return cli.Transcribe{
		Transcribe: cfg.Transcribe(db.Sources(), models),
		Cut:        cutFn,
		Close: func() error {
			_ = closeModels()
			return db.Close()
		},
	}, nil
}

// indexedCut is the index and the extraction of a vault's sources over it.
type indexedCut[C any] struct {
	db  *container.Index
	cut C
}

// createCut opens the index and makes the extraction over it. The index is
// closed when the extraction cannot be made.
func createCut[C any](
	ctx context.Context, cfg container.Config, v domain.Vault,
	extract func(port.SourceRepository, port.SourceQueries, domain.Vault) (C, error),
) (indexedCut[C], error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return indexedCut[C]{}, err
	}
	cut, err := extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		_ = db.Close()
		return indexedCut[C]{}, err
	}
	return indexedCut[C]{db: db, cut: cut}, nil
}

// createImportURL is the download of an address into the vault.
func createImportURL(ctx context.Context, cfg container.Config) (cli.ImportURL, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.ImportURL{}, err
	}
	return cli.ImportURL{ImportURL: cfg.ImportURL(ctx, db, cfg.Downloader(ctx)), Close: db.Close}, nil
}

// createProofreadRecognition is the proofreading of recognized scans, when it is
// configured.
func createProofreadRecognition(
	ctx context.Context, cfg container.Config, v domain.Vault,
) (cli.ProofreadRecognition, error) {
	cfg.SetPluginVaultPath(v.Path)
	proofread, target, err := cfg.CreateRecognitionProofread()
	if err != nil || !target {
		return cli.ProofreadRecognition{}, err
	}
	indexed, err := createCut(ctx, cfg, v, cfg.Extract)
	if err != nil {
		return cli.ProofreadRecognition{}, err
	}
	return cli.ProofreadRecognition{
		Proofread: proofread, Cut: indexed.cut, IsHeld: true, Close: indexed.db.Close,
	}, nil
}

// createProofreadTranscript is the proofreading of transcripts, when it is
// configured.
func createProofreadTranscript(
	ctx context.Context, cfg container.Config, v domain.Vault,
) (cli.ProofreadTranscript, error) {
	cfg.SetPluginVaultPath(v.Path)
	proofread, target, err := cfg.CreateTranscriptProofread()
	if err != nil || !target {
		return cli.ProofreadTranscript{}, err
	}
	indexed, err := createCut(ctx, cfg, v, cfg.Extract)
	if err != nil {
		return cli.ProofreadTranscript{}, err
	}
	return cli.ProofreadTranscript{
		Proofread: proofread, Cut: indexed.cut, IsHeld: true, Close: indexed.db.Close,
	}, nil
}

// createSearch is the search over the index. Only the provider that embeds
// questions is opened, because nothing a search does fills an index.
func createSearch(
	ctx context.Context, cfg container.Config, errorHandler port.ErrorHandler,
) (cli.Search, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Search{}, err
	}
	asking, closeAsking, why := cfg.Embedder(ctx)
	return cli.Search{
		Search: cfg.NewSearch(db, asking, errorHandler),
		Words:  why,
		Close: func() error {
			closeIfOpen(closeAsking)
			return db.Close()
		},
	}, nil
}

// createProblems is the problems the index holds.
func createProblems(ctx context.Context, cfg container.Config) (cli.Problems, error) {
	db, err := cfg.OpenIndex(ctx)
	if err != nil {
		return cli.Problems{}, err
	}
	return cli.Problems{Problems: db.Problems(), Close: db.Close}, nil
}
