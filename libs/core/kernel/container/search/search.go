package search

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	searchusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// NewSearch is the search a question is answered by, put together the one way:
// the passages, the vault's files, the text read out of books, and the model a
// question is embedded by. Its floor is the one the vault has under that model.
func (c Config) NewSearch(db *vault.Index, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	policy := c.NewFloorPolicy(db, asking, c.ErrorHandler)
	return c.NewSearchOver(db.Passages(), asking, errorHandler).WithFloorSource(policy.GetFloor)
}

// NewSearchOver is that search over the passages given, for a run that holds
// the index open for asking alone. Its floor is the model's own, then
// search.DefaultFloor.
func (c Config) NewSearchOver(passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) searchusecase.Search {
	policy := searchusecase.NewFloorPolicy(nil, GetFloorHint(asking), nil)
	return searchusecase.New(passages, c.VaultReaders, c.TextReaders, asking, NewEmbedQueries(asking), errorHandler).
		WithExtractors(c.ResolvedCompute.Workers).
		WithFloorSource(policy.GetFloor)
}
