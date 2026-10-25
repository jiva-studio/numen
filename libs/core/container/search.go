package container

import (
	"github.com/jiva-studio/numen/libs/core/usecase/search"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// NewSearch is the search a question is answered by, put together the one way:
// the passages, the vault's files, the text read out of books, and the model a
// question is embedded by.
func (c Config) NewSearch(db *Index, asking embeddingport.Embedder, errorHandler func(error)) search.Search {
	return c.NewSearchOver(db.Passages(), asking, errorHandler)
}

// NewSearchOver is that search over the passages given, for a run that holds
// the index open for asking alone.
func (c Config) NewSearchOver(passages sourceport.PassageQueries, asking embeddingport.Embedder, errorHandler func(error)) search.Search {
	return search.New(passages, c.VaultReaders(), c.getTextReaders(),
		asking, c.Embedding.Floor, errorHandler).WithExtractors(c.ResolvedCompute.Workers)
}
