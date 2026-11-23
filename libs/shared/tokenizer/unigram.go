package tokenizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"unicode/utf8"
)

// unknownPenalty is how far below the lowest piece score an unknown rune is scored, as K_UNK_PENALTY
// of Hugging Face `tokenizers`.
const unknownPenalty = 10

// vocabulary is a Unigram model: the id of a piece is its index in model.vocab.
type vocabulary struct {
	ids          map[string]int
	scores       []float64
	maxPieceLen  int
	unkID        int
	unknownScore float64
}

func newVocabulary(model unigramModel) (*vocabulary, error) {
	if model.UnkID == nil {
		return nil, errors.New("unk_id is missing")
	}
	if model.ByteFallback {
		return nil, errors.New("byte_fallback is not supported")
	}
	v := &vocabulary{
		ids:    make(map[string]int, len(model.Vocab)),
		scores: make([]float64, len(model.Vocab)),
		unkID:  *model.UnkID,
	}
	if v.unkID < 0 || v.unkID >= len(model.Vocab) {
		return nil, fmt.Errorf("unk_id %d is outside the vocabulary of %d pieces", v.unkID, len(model.Vocab))
	}
	lowest := math.Inf(1)
	for id, entry := range model.Vocab {
		if len(entry) != 2 {
			return nil, fmt.Errorf("vocab entry %d holds %d values, want a piece and a score", id, len(entry))
		}
		var piece string
		if err := json.Unmarshal(entry[0], &piece); err != nil {
			return nil, fmt.Errorf("vocab entry %d: %w", id, err)
		}
		if err := json.Unmarshal(entry[1], &v.scores[id]); err != nil {
			return nil, fmt.Errorf("vocab entry %d: %w", id, err)
		}
		v.ids[piece] = id
		v.maxPieceLen = max(v.maxPieceLen, len(piece))
		lowest = min(lowest, v.scores[id])
	}
	v.unknownScore = lowest - unknownPenalty
	return v, nil
}

// latticeNode is the best path found so far to one byte offset of a piece.
type latticeNode struct {
	score float64
	start int
	id    int
	isSet bool
}

func updateBestNode(best []latticeNode, start, end, id int, score float64) {
	total := best[start].score + score
	if !best[end].isSet || total > best[end].score {
		best[end] = latticeNode{score: total, start: start, id: id, isSet: true}
	}
}

func (v *vocabulary) appendIDs(out []int, piece string) []int {
	if piece == "" {
		return out
	}
	best := make([]latticeNode, len(piece)+1)
	best[0].isSet = true
	for start := 0; start < len(piece); {
		_, firstLen := utf8.DecodeRuneInString(piece[start:])
		if best[start].isSet {
			v.updateBestFrom(best, piece, start, firstLen)
		}
		start += firstLen
	}
	return v.appendPath(out, best)
}

func (v *vocabulary) updateBestFrom(best []latticeNode, piece string, start, firstLen int) {
	hasSingle := false
	for end := start + firstLen; end <= len(piece) && end-start <= v.maxPieceLen; {
		if id, ok := v.ids[piece[start:end]]; ok {
			hasSingle = hasSingle || end-start == firstLen
			updateBestNode(best, start, end, id, v.scores[id])
		}
		if end == len(piece) {
			break
		}
		_, size := utf8.DecodeRuneInString(piece[end:])
		end += size
	}
	if !hasSingle {
		updateBestNode(best, start, start+firstLen, v.unkID, v.unknownScore)
	}
}

func (v *vocabulary) appendPath(out []int, best []latticeNode) []int {
	var reversed []int
	for end := len(best) - 1; end > 0; end = best[end].start {
		id := best[end].id
		if id == v.unkID && len(reversed) > 0 && reversed[len(reversed)-1] == v.unkID {
			continue
		}
		reversed = append(reversed, id)
	}
	for _, id := range slices.Backward(reversed) {
		out = append(out, id)
	}
	return out
}
