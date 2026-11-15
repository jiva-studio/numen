package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// QueryParser parses the text of a query against a catalog. A query that does not parse or type-check
// returns *searchquery.ParseError with every problem; another error is a failure of the parser.
type QueryParser interface {
	ParseQuery(text string, catalog searchquery.Catalog) (searchquery.RawNode, error)
}

// DocumentQueries lists the documents of a vault within a scope superset and reads its word index.
type DocumentQueries interface {
	WordQueries
	// ListDocuments returns the sources of one vault under any of q.PathPrefixes, of q.SourceKinds and at q.Paths,
	// with their note title, size, time of change and the count of their small chunks that hold a vector under
	// q.Recipe, ordered by path, at most q.Limit+1 of them, all of them when q.Limit is zero.
	ListDocuments(ctx context.Context, vaultID domain.VaultID, q searchquery.DocumentQuery) ([]searchquery.Document, error)
}

// VectorQueries reads vectors for a search by meaning.
type VectorQueries interface {
	// FindCoarseCandidates returns the chunks whose one-bit vectors are nearest the query, nearest first.
	FindCoarseCandidates(ctx context.Context, vaultID domain.VaultID, q source.CoarseQuery) ([]domain.ChunkID, error)
	// ListDocumentVectors returns the vectors under q.Recipe of the small chunks of the documents at q.Paths,
	// at most q.Limit after chunk q.After, ordered by chunk id.
	ListDocumentVectors(ctx context.Context, vaultID domain.VaultID, q searchquery.DocumentVectorQuery) ([]source.StoredVector, error)
	// ListStoredVectors returns the full-precision vectors of the chunks under q.Recipe.
	ListStoredVectors(ctx context.Context, q source.StoredVectorQuery) ([]source.StoredVector, error)
}

// WordQueries reads the word index for a words condition.
type WordQueries interface {
	// FindWords returns the small chunks whose text matches the FTS5 expression, with their negated bm25,
	// best first, within q.Paths when set, else within q.PathPrefixes and q.SourceKinds, at most q.Limit. An
	// expression FTS5 cannot parse returns source.ErrInvalidWords.
	FindWords(ctx context.Context, vaultID domain.VaultID, q searchquery.WordQuery) ([]searchquery.WordMatch, error)
	// GetWordScores returns the negated bm25 of each given chunk that matches the expression.
	GetWordScores(ctx context.Context, vaultID domain.VaultID, expression string, chunks []domain.ChunkID) (map[domain.ChunkID]float64, error)
}

// NoteContents reads what the frontmatter and the headings of the notes of a vault hold.
type NoteContents interface {
	// ListPropertyValues returns the path and the value of every note of the vault whose stored frontmatter
	// has the top-level key. A note without frontmatter or with unreadable frontmatter is absent.
	ListPropertyValues(ctx context.Context, vaultID domain.VaultID, key string) ([]searchquery.PropertyEntry, error)
	// FindHeadings returns the headings, with the path of their note, whose words hold every word of the text,
	// and every heading of the vault when the text has no word.
	FindHeadings(ctx context.Context, vaultID domain.VaultID, text string) ([]searchquery.HeadingEntry, error)
	// CountUnreadableFrontmatter counts the notes of the vault whose frontmatter could not be read.
	CountUnreadableFrontmatter(ctx context.Context, vaultID domain.VaultID) (int, error)
}

// LinkReaders reads the links of a vault for the link functions.
type LinkReaders interface {
	noteport.LinkBatches
	// GetNoteCandidates returns the paths of the notes of the vault an address could mean, ordered by path:
	// for a note address the note with that identifier in this vault, for a name the notes the link
	// resolution would consider.
	GetNoteCandidates(ctx context.Context, vaultID domain.VaultID, address domain.Address) ([]string, error)
	// ListFilePaths returns the paths of the sources of the vault that are not notes, ordered by path.
	ListFilePaths(ctx context.Context, vaultID domain.VaultID) ([]string, error)
	// ListLinksByName returns the links written in the notes of the vault whose target's last segment folds
	// to the name, with the path of the note each is written in and without resolution.
	ListLinksByName(ctx context.Context, vaultID domain.VaultID, foldedName string) ([]domain.ResolvedLink, error)
}
