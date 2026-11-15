package search_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/wordmatch"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	sourcedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const longNote = "notes/Memoir.md"

func getMemoir() string {
	return "Meadow paths " + strings.Repeat("filler ", 340) + "childhood memories " + strings.Repeat("filler ", 80) + "\n"
}

func (c corpus) score(t *testing.T, v domain.Vault, model sourcefake.Embedder, floors *memoryFloors, q search.ChunkScoreQuery) ([]search.ChunkScore, error) {
	t.Helper()
	deps := search.ScoreChunksDeps{
		Places:  c.db.ChunkSearch(),
		Words:   wordmatch.NewMatcher(),
		Readers: vaultreader.Readers{},
		Floors:  search.NewFloorPolicy(floors, nil, nil),
	}
	if model != nil {
		deps.Source = sourcefake.Wrap(model)
	}
	return search.NewScoreChunks(deps).Execute(t.Context(), v, q)
}

func (c corpus) findChunkID(t *testing.T, v domain.Vault, words string) domain.ChunkID {
	t.Helper()
	found := c.searchIn(t, v, words, nil)
	if len(found) == 0 {
		t.Fatalf("the words %q found nothing", words)
	}
	return found[0].ChunkID
}

func newWordsCondition(query string) search.ScoreCondition {
	return search.ScoreCondition{Words: query}
}

func newMeaningCondition(text string) search.ScoreCondition {
	return search.ScoreCondition{Meaning: text}
}

func TestScoreChunksMatchesAWordPastTheFirst1200BytesOfTheChunk(t *testing.T) {
	c := newCorpus(t)
	v := c.addVault(t, map[string]string{longNote: getMemoir()})
	c.cutBySize(t, v, longNote, 300)
	if at := strings.Index(getMemoir(), "childhood"); at < 2000 {
		t.Fatalf("the word is at byte %d, want it past the excerpt", at)
	}

	results, err := search.NewSearchText(c.search(nil)).Execute(t.Context(), v, search.SearchResultQuery{Query: "meadow"})
	if err != nil || len(results) != 1 {
		t.Fatalf("the search found %v, %v, want the memoir", sources(results), err)
	}
	if strings.Contains(results[0].Text, "childhood") {
		t.Fatal("the excerpt of the result holds the word, so this proves nothing")
	}

	got, err := c.score(t, v, nil, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs:   []domain.ChunkID{results[0].ChunkID},
		Conditions: []search.ScoreCondition{newWordsCondition("childhood"), newWordsCondition("zzqqxx")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Conditions) != 2 {
		t.Fatalf("scored %+v, want one chunk with two conditions", got)
	}
	if !got[0].Conditions[0].IsMatch {
		t.Error("a word past the excerpt does not match")
	}
	if got[0].Conditions[1].IsMatch {
		t.Error("a word the chunk does not have matches")
	}
	if got[0].Conditions[0].Similarity != nil {
		t.Errorf("a words condition has the similarity %v, want none", *got[0].Conditions[0].Similarity)
	}
}

func TestScoreChunksMatchesTwoWordsThatAreInDifferentSmallChunksOfOneChunk(t *testing.T) {
	c := newCorpus(t)
	v := c.addVault(t, map[string]string{longNote: "Heron " + strings.Repeat("filler ", 100) + "reed\n"})
	c.cutBySize(t, v, longNote, 300)

	both, err := queriesOf(c.db).FindLexical(t.Context(), v.ID, sourcedomain.LexicalQuery{Text: "heron reed", Limit: 10})
	if err != nil || len(both) != 0 {
		t.Fatalf("one small chunk holds both words (%d, %v), so this proves nothing", len(both), err)
	}

	got, err := c.score(t, v, nil, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs:   []domain.ChunkID{c.findChunkID(t, v, "heron")},
		Conditions: []search.ScoreCondition{newWordsCondition("heron reed"), newWordsCondition(`"heron reed"`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Conditions[0].IsMatch {
		t.Errorf("scored %+v, want the two words of one chunk to match", got)
	}
	if got[0].Conditions[1].IsMatch {
		t.Error("the phrase matches two words that are not next to each other")
	}
}

func TestScoreChunksAnswersWordsConditionsWithoutAModel(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")

	got, err := c.score(t, c.first, nil, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{newWordsCondition("disorder NOT heat")},
	})
	if err != nil || len(got) != 1 || !got[0].Conditions[0].IsMatch {
		t.Errorf("an installation with no model scored %+v, %v, want the words to match", got, err)
	}
}

func TestScoreChunksRejectsAMeaningConditionWithoutAModel(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")

	_, err := c.score(t, c.first, nil, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{newWordsCondition("disorder"), newMeaningCondition("chaos")},
	})
	if !errors.Is(err, plugin.ErrNoModel) {
		t.Errorf("a meaning condition with no model returned %v, want ErrNoModel", err)
	}
}

func TestScoreChunksReturnsForAChunkTheSimilarityTheMeaningSearchReturnedForIt(t *testing.T) {
	c := newCorpus(t)
	v := c.addVault(t, map[string]string{longNote: getMemoir()})
	c.cutBySize(t, v, longNote, 1500)
	c.vectorizePath(t, v, longNote, func(record chunk.Record) []float32 {
		if record.Start == 0 {
			return newNearVector()
		}
		return newVector(+1)
	})
	model := oneWay{direction: newVector(+1)}

	found, err := c.search(model).Execute(t.Context(), v, "a quiet morning", search.GetTypingParameters(search.Sources, 10))
	if err != nil || len(found) != 1 || found[0].Similarity == nil {
		t.Fatalf("the meaning search found %+v, %v, want the memoir with a similarity", found, err)
	}
	if best := vector.SimilarityBytes(newVector(+1), precise(newVector(+1))); *found[0].Similarity != best {
		t.Fatalf("the meaning search returned %v, want the best small chunk, %v", *found[0].Similarity, best)
	}

	got, err := c.score(t, v, model, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs: []domain.ChunkID{found[0].ChunkID}, Conditions: []search.ScoreCondition{newMeaningCondition("a quiet morning")},
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("scored %+v, %v, want the memoir", got, err)
	}
	if one := got[0].Conditions[0]; one.Similarity == nil || *one.Similarity != *found[0].Similarity || !one.IsMatch {
		t.Errorf("the chunk scored %+v, want the similarity of the search, %v, and a match", one, *found[0].Similarity)
	}
}

func TestScoreChunksReportsASimilarityBelowTheFloorOfTheVaultAsNoMatch(t *testing.T) {
	c := newCorpus(t)
	c.vectorizePath(t, c.first, "notes/Entropy.md", func(chunk.Record) []float32 { return newVector(-1) })
	id := c.findChunkID(t, c.first, "disorder")

	got, err := c.score(t, c.first, oneWay{direction: newVector(+1)}, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{newMeaningCondition("order")},
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("scored %+v, %v, want the note", got, err)
	}
	one := got[0].Conditions[0]
	if one.Similarity == nil || *one.Similarity > -0.9 {
		t.Errorf("the similarity is %v, want the opposite direction, about -1", one.Similarity)
	}
	if one.IsMatch {
		t.Error("a similarity below the floor matches")
	}
}

func TestScoreChunksUsesTheStoredFloorOfTheVault(t *testing.T) {
	c := newCorpus(t)
	c.vectorizePath(t, c.first, "notes/Entropy.md", func(chunk.Record) []float32 { return newNearVector() })
	id := c.findChunkID(t, c.first, "disorder")
	model := oneWay{direction: newVector(+1)}
	floors := newMemoryFloors()
	query := search.ChunkScoreQuery{ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{newMeaningCondition("order")}}

	byDefault, err := c.score(t, c.first, model, floors, query)
	if err != nil || len(byDefault) != 1 || !byDefault[0].Conditions[0].IsMatch {
		t.Fatalf("under the default floor scored %+v, %v, want a match at about 0.75", byDefault, err)
	}

	if err := floors.WriteFloor(t.Context(), c.first, getModel().Recipe(), 0.9); err != nil {
		t.Fatal(err)
	}
	stored, err := c.score(t, c.first, model, floors, query)
	if err != nil || len(stored) != 1 || stored[0].Conditions[0].IsMatch {
		t.Errorf("under a stored floor of 0.9 scored %+v, %v, want no match", stored, err)
	}
}

func TestScoreChunksReturnsNoSimilarityForAChunkWithNoSmallChunkHavingAVector(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")

	got, err := c.score(t, c.first, oneWay{direction: newVector(+1)}, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{newMeaningCondition("order")},
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("scored %+v, %v, want the note", got, err)
	}
	if one := got[0].Conditions[0]; one.Similarity != nil || one.IsMatch {
		t.Errorf("a chunk with no vector scored %+v, want no similarity and no match", one)
	}
}

func TestScoreChunksOmitsAChunkIDTheVaultDoesNotHold(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")
	otherVault := c.findChunkID(t, c.second, "beacon")

	got, err := c.score(t, c.first, nil, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs:   []domain.ChunkID{"999999999", otherVault, id, id},
		Conditions: []search.ScoreCondition{newWordsCondition("disorder")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ChunkID != id || !got[0].Conditions[0].IsMatch {
		t.Errorf("scored %+v, want only the chunk of the vault, once", got)
	}
}

func TestScoreChunksRejectsMoreThan500Chunks(t *testing.T) {
	c := newCorpus(t)
	ids := make([]domain.ChunkID, 501)
	for i := range ids {
		ids[i] = domain.ChunkID(strconv.Itoa(900000000 + i))
	}
	query := search.ChunkScoreQuery{ChunkIDs: ids, Conditions: []search.ScoreCondition{newWordsCondition("disorder")}}

	if _, err := c.score(t, c.first, nil, newMemoryFloors(), query); !errors.Is(err, plugin.ErrLimitExceeded) {
		t.Errorf("501 chunks returned %v, want ErrLimitExceeded", err)
	}
	query.ChunkIDs = ids[:500]
	if _, err := c.score(t, c.first, nil, newMemoryFloors(), query); err != nil {
		t.Errorf("500 chunks returned %v, want an answer", err)
	}
}

func TestScoreChunksRejectsMoreThan16Conditions(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")
	conditions := make([]search.ScoreCondition, 17)
	for i := range conditions {
		conditions[i] = newWordsCondition("disorder")
	}
	query := search.ChunkScoreQuery{ChunkIDs: []domain.ChunkID{id}, Conditions: conditions}

	if _, err := c.score(t, c.first, nil, newMemoryFloors(), query); !errors.Is(err, plugin.ErrLimitExceeded) {
		t.Errorf("17 conditions returned %v, want ErrLimitExceeded", err)
	}
	query.Conditions = conditions[:16]
	if got, err := c.score(t, c.first, nil, newMemoryFloors(), query); err != nil || len(got) != 1 || len(got[0].Conditions) != 16 {
		t.Errorf("16 conditions scored %+v, %v, want one chunk with 16 scores", got, err)
	}
}

func TestScoreChunksRejectsAConditionTextOver8192Bytes(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")
	long := strings.Repeat("a", 8193)

	for _, condition := range []search.ScoreCondition{newWordsCondition(long), newMeaningCondition(long)} {
		_, err := c.score(t, c.first, oneWay{direction: newVector(+1)}, newMemoryFloors(), search.ChunkScoreQuery{
			ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{condition},
		})
		if !errors.Is(err, plugin.ErrLimitExceeded) {
			t.Errorf("a condition of 8193 bytes returned %v, want ErrLimitExceeded", err)
		}
	}
}

func TestScoreChunksRejectsAConditionWithNoTextOrBothTexts(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")

	for name, condition := range map[string]search.ScoreCondition{
		"no text":    {},
		"both texts": {Meaning: "order", Words: "disorder"},
	} {
		_, err := c.score(t, c.first, oneWay{direction: newVector(+1)}, newMemoryFloors(), search.ChunkScoreQuery{
			ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{condition},
		})
		if !errors.Is(err, plugin.ErrInvalidCondition) {
			t.Errorf("a condition with %s returned %v, want ErrInvalidCondition", name, err)
		}
	}
}

func TestScoreChunksReturnsErrInvalidWordsForAQueryFTS5CannotParse(t *testing.T) {
	c := newCorpus(t)
	id := c.findChunkID(t, c.first, "disorder")

	_, err := c.score(t, c.first, nil, newMemoryFloors(), search.ChunkScoreQuery{
		ChunkIDs: []domain.ChunkID{id}, Conditions: []search.ScoreCondition{newWordsCondition(`"disorder AND`)},
	})
	if !errors.Is(err, sourcedomain.ErrInvalidWords) {
		t.Errorf("a words query FTS5 cannot parse returned %v, want ErrInvalidWords", err)
	}
}
