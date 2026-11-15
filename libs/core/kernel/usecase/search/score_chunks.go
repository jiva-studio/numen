package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// The limits of one ScoreChunks call.
const (
	MaxScoredChunks     = 500
	MaxScoredConditions = 16
	MaxConditionBytes   = 8192
)

// ScoreCondition is one condition a chunk is scored against: a text compared by meaning or an FTS5
// query matched against the words. Exactly one of the two is set.
type ScoreCondition struct {
	Meaning string
	Words   string
}

// ChunkScoreQuery is the chunks to score and the conditions to score them against.
type ChunkScoreQuery struct {
	ChunkIDs   []domain.ChunkID
	Conditions []ScoreCondition
}

// ConditionScore is how one chunk scores against one condition. Similarity is set for a meaning
// condition when a small chunk inside the chunk carries a vector.
type ConditionScore struct {
	Similarity *float64
	IsMatch    bool
}

// ChunkScore is the score of one chunk against each condition, in the order of the conditions.
type ChunkScore struct {
	ChunkID    domain.ChunkID
	Conditions []ConditionScore
}

// ScoreChunksDeps is what ScoreChunks reads. A nil Source is an installation with no model, and a nil
// Texts reads the text of notes only.
type ScoreChunksDeps struct {
	Places  sourceport.ChunkPlaces
	Words   sourceport.WordMatching
	Readers vaultport.VaultReaders
	Texts   textdomain.Readers
	Source  searchsourceport.QuerySource
	Floors  FloorPolicy
}

// ScoreChunks scores chunks of a vault against conditions over the whole large chunk that encloses each.
type ScoreChunks struct {
	places  sourceport.ChunkPlaces
	words   sourceport.WordMatching
	readers vaultport.VaultReaders
	texts   textdomain.Readers
	source  searchsourceport.QuerySource
	floors  FloorPolicy
}

// NewScoreChunks creates the scoring over the index, the word matcher and the search source.
func NewScoreChunks(deps ScoreChunksDeps) *ScoreChunks {
	return &ScoreChunks{
		places: deps.Places, words: deps.Words, readers: deps.Readers,
		texts: deps.Texts, source: deps.Source, floors: deps.Floors,
	}
}

// Execute returns one score per distinct chunk of q.ChunkIDs the vault holds, in the order given. A
// meaning condition is embedded once and compared with the vector of every small chunk inside the
// chunk; the best similarity matches at or above the floor of the vault. A words condition matches when
// the whole text of the chunk matches it. More than MaxScoredChunks chunks, MaxScoredConditions
// conditions or a text over MaxConditionBytes returns plugin.ErrLimitExceeded, a malformed condition
// plugin.ErrInvalidCondition, a meaning condition with no model plugin.ErrNoModel, and a words query
// FTS5 cannot parse source.ErrInvalidWords.
func (u *ScoreChunks) Execute(ctx context.Context, v domain.Vault, q ChunkScoreQuery) ([]ChunkScore, error) {
	if err := checkScoreQuery(q); err != nil {
		return nil, err
	}
	if hasMeaningCondition(q.Conditions) && !u.hasModel() {
		return nil, plugin.ErrNoModel
	}
	places, err := u.places.GetEnclosingResults(ctx, v.ID, getDistinctChunks(q.ChunkIDs))
	if err != nil || len(places) == 0 {
		return nil, err
	}
	out := make([]ChunkScore, len(places))
	for i, p := range places {
		out[i] = ChunkScore{ChunkID: p.ChunkID, Conditions: make([]ConditionScore, len(q.Conditions))}
	}
	if err := u.scoreMeaning(ctx, v, places, q.Conditions, out); err != nil {
		return nil, err
	}
	if err := u.scoreWords(ctx, v, places, q.Conditions, out); err != nil {
		return nil, err
	}
	return out, nil
}

func checkScoreQuery(q ChunkScoreQuery) error {
	if len(q.ChunkIDs) > MaxScoredChunks {
		return fmt.Errorf("%w: %d chunks, at most %d", plugin.ErrLimitExceeded, len(q.ChunkIDs), MaxScoredChunks)
	}
	if len(q.Conditions) > MaxScoredConditions {
		return fmt.Errorf("%w: %d conditions, at most %d", plugin.ErrLimitExceeded, len(q.Conditions), MaxScoredConditions)
	}
	for i, c := range q.Conditions {
		if len(c.Meaning) > MaxConditionBytes || len(c.Words) > MaxConditionBytes {
			return fmt.Errorf("%w: condition %d is over %d bytes", plugin.ErrLimitExceeded, i, MaxConditionBytes)
		}
		if (c.Meaning != "") == (c.Words != "") || strings.TrimSpace(c.Meaning+c.Words) == "" {
			return fmt.Errorf("%w: condition %d", plugin.ErrInvalidCondition, i)
		}
	}
	return nil
}

func hasMeaningCondition(conditions []ScoreCondition) bool {
	for _, c := range conditions {
		if c.Meaning != "" {
			return true
		}
	}
	return false
}

func (u *ScoreChunks) hasModel() bool {
	return u.source != nil && u.source.Describe().IsDescribed()
}

func getDistinctChunks(ids []domain.ChunkID) []domain.ChunkID {
	seen := make(map[domain.ChunkID]bool, len(ids))
	out := make([]domain.ChunkID, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (u *ScoreChunks) scoreMeaning(ctx context.Context, v domain.Vault, places []domain.SearchResult, conditions []ScoreCondition, out []ChunkScore) error {
	if !hasMeaningCondition(conditions) {
		return nil
	}
	recipe := u.source.Describe().Recipe()
	smalls, vectors, err := u.readSmallVectors(ctx, v.ID, recipe, places)
	if err != nil {
		return err
	}
	floor := u.floors.GetFloor(ctx, v, recipe)
	for ci, c := range conditions {
		if c.Meaning == "" {
			continue
		}
		query, err := u.embed(ctx, c.Meaning)
		if err != nil {
			return err
		}
		best, err := getBestSimilarities(query, smalls, vectors)
		if err != nil {
			return err
		}
		for i, p := range places {
			if similarity, has := best[p.ChunkID]; has {
				out[i].Conditions[ci] = ConditionScore{Similarity: &similarity, IsMatch: similarity >= floor}
			}
		}
	}
	return nil
}

func (u *ScoreChunks) readSmallVectors(
	ctx context.Context, vaultID domain.VaultID, recipe string, places []domain.SearchResult,
) ([]source.SmallChunk, map[domain.ChunkID][]byte, error) {
	ids := make([]domain.ChunkID, len(places))
	for i, p := range places {
		ids[i] = p.ChunkID
	}
	smalls, err := u.places.ListSmallChunks(ctx, vaultID, ids)
	if err != nil || len(smalls) == 0 {
		return nil, nil, err
	}
	smallIDs := make([]domain.ChunkID, 0, len(smalls))
	for _, one := range smalls {
		smallIDs = append(smallIDs, one.Small)
	}
	stored, err := u.places.ListStoredVectors(ctx, source.StoredVectorQuery{Recipe: recipe, Chunks: getDistinctChunks(smallIDs)})
	if err != nil {
		return nil, nil, err
	}
	vectors := make(map[domain.ChunkID][]byte, len(stored))
	for _, one := range stored {
		vectors[one.Chunk] = one.Vector
	}
	return smalls, vectors, nil
}

func (u *ScoreChunks) embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := u.source.EmbedQueries(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", plugin.ErrEmbeddingFailed, err)
	}
	if len(vectors) != 1 {
		return nil, fmt.Errorf("%w: %d vectors for one text", plugin.ErrEmbeddingFailed, len(vectors))
	}
	return vectors[0], nil
}

func getBestSimilarities(query []float32, smalls []source.SmallChunk, vectors map[domain.ChunkID][]byte) (map[domain.ChunkID]float64, error) {
	best := map[domain.ChunkID]float64{}
	for _, one := range smalls {
		stored, has := vectors[one.Small]
		if !has {
			continue
		}
		if len(stored) != len(query) {
			return nil, fmt.Errorf("chunk %s holds %d dimensions where the query has %d", one.Small, len(stored), len(query))
		}
		similarity := vector.SimilarityBytes(query, stored)
		if current, found := best[one.Chunk]; !found || similarity > current {
			best[one.Chunk] = similarity
		}
	}
	return best, nil
}

func (u *ScoreChunks) scoreWords(ctx context.Context, v domain.Vault, places []domain.SearchResult, conditions []ScoreCondition, out []ChunkScore) error {
	if !hasWordsCondition(conditions) {
		return nil
	}
	texts, err := u.readTexts(ctx, v, places)
	if err != nil {
		return err
	}
	for ci, c := range conditions {
		if c.Words == "" {
			continue
		}
		matched, err := u.words.MatchWords(ctx, c.Words, texts)
		if err != nil {
			return err
		}
		for i := range places {
			out[i].Conditions[ci] = ConditionScore{IsMatch: matched[i]}
		}
	}
	return nil
}

func hasWordsCondition(conditions []ScoreCondition) bool {
	for _, c := range conditions {
		if c.Words != "" {
			return true
		}
	}
	return false
}

func (u *ScoreChunks) readTexts(ctx context.Context, v domain.Vault, places []domain.SearchResult) ([]string, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}
	of := sourceTexts{files: reader}
	if u.texts != nil {
		if of.texts, err = u.texts.Open(v); err != nil {
			return nil, err
		}
	}
	texts := make([]string, len(places))
	groups, order := groupByPath(places)
	for _, path := range order {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		indices := groups[path]
		prose, _, err := of.getText(ctx, path, places[indices[0]].VersionID)
		if edit.IsNoNote(err) || errors.Is(err, errUnreadable) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		for _, i := range indices {
			texts[i] = span(prose, places[i].Start, places[i].Length)
		}
	}
	return texts, nil
}
