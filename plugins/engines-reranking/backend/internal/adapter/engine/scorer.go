package engine

import (
	"cmp"
	"context"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"runtime"
	"slices"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/libs/shared/tokenizer"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

// The inputs a cross-encoder exported to ONNX takes.
const (
	inputIDs      = "input_ids"
	attentionMask = "attention_mask"
	tokenTypeIDs  = "token_type_ids"
)

// outputLogits is the output a cross-encoder exported to ONNX answers with.
const outputLogits = "logits"

// Scorer is one cross-encoder, loaded.
type Scorer struct {
	name     string
	pairs    *pairTokenizer
	isTyped  bool
	output   string
	sessions *sessionPool[*ort.Session]
	paddings *sync.Pool
}

// Open loads the model of p, fetching its files first where this machine does not hold them. It fails on a
// graph that takes an input other than the tokens, the mask and the token types, or answers more than one
// logit for a pair.
func Open(ctx context.Context, library *onnx.Library, p profile.Profile, fetch Fetch, queue host.InferenceQueue) (*Scorer, error) {
	paths, err := Locate(ctx, p, fetch)
	if err != nil {
		return nil, err
	}
	return host.OpenModel(ctx, queue, func() (*Scorer, error) { return openModel(ctx, library, p, paths) })
}

func openModel(ctx context.Context, library *onnx.Library, p profile.Profile, paths onnx.ModelFiles) (*Scorer, error) {
	pairs, err := loadPairTokenizer(paths.Tokenizer, p.MaxTokens)
	if err != nil {
		return nil, err
	}
	opened, err := library.Open(ctx, onnx.GetLibrary())
	if err != nil {
		return nil, err
	}
	engine := opened.Engine
	settings := getSessionSettings(paths.Model)
	options, _, err := library.NewSessionOptions(engine, settings)
	if err != nil {
		return nil, err
	}
	defer options.Destroy()
	session, err := onnx.NewSession(engine, settings.CacheRoot, paths.Model, options)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", paths.Model, err)
	}
	s, err := newScorer(p.ID, pairs, session)
	if err != nil {
		session.Destroy()
		return nil, err
	}
	if _, err := s.ScorePairs(ctx, "", []string{""}); err != nil {
		s.sessions.close()
		return nil, fmt.Errorf("%s: %w", p.ID, err)
	}
	return s, nil
}

func newScorer(name string, pairs *pairTokenizer, session *ort.Session) (*Scorer, error) {
	inputs, err := checkInputs(name, session.InputNames)
	if err != nil {
		return nil, err
	}
	output, err := chooseOutput(session.OutputNames)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &Scorer{
		name:     name,
		pairs:    pairs,
		isTyped:  inputs.isTyped,
		output:   output,
		sessions: newSessionPool([]*ort.Session{session}, 1, (*ort.Session).Destroy),
		paddings: newPaddingPool(),
	}, nil
}

// modelInputs says which of the optional inputs a model takes.
type modelInputs struct {
	isTyped bool
}

func checkInputs(owner string, names []string) (modelInputs, error) {
	var inputs modelInputs
	for _, name := range names {
		switch name {
		case inputIDs, attentionMask:
		case tokenTypeIDs:
			inputs.isTyped = true
		default:
			return modelInputs{}, fmt.Errorf("%s requires an unsupported input: %s", owner, name)
		}
	}
	if !slices.Contains(names, inputIDs) || !slices.Contains(names, attentionMask) {
		return modelInputs{}, fmt.Errorf("%s takes %v, expected tokens and mask", owner, names)
	}
	return inputs, nil
}

func chooseOutput(names []string) (string, error) {
	if slices.Contains(names, outputLogits) {
		return outputLogits, nil
	}
	if len(names) == 1 {
		return names[0], nil
	}
	return "", fmt.Errorf("answers with %v, expected %s", names, outputLogits)
}

// Close releases the sessions of the model.
func (s *Scorer) Close() error {
	s.sessions.close()
	return nil
}

// ScorePairs returns the score of each document against query, in the order of the documents.
func (s *Scorer) ScorePairs(ctx context.Context, query string, documents []string) ([]float32, error) {
	encoded := make([]tokenizer.Encoding, len(documents))
	for i, document := range documents {
		enc, err := s.pairs.encode(query, document)
		if err != nil {
			return nil, fmt.Errorf("encode document %d: %w", i, err)
		}
		encoded[i] = enc
	}
	return scorePairs(ctx, encoded, s.pairs.maxTokens, s.forward)
}

func scorePairs(
	ctx context.Context,
	pairs []tokenizer.Encoding,
	limit int,
	pass func(context.Context, []tokenizer.Encoding) ([]float32, error),
) ([]float32, error) {
	order := make([]int, len(pairs))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(len(pairs[a].IDs), len(pairs[b].IDs)) })

	out := make([]float32, len(pairs))
	batch := make([]tokenizer.Encoding, 0, batchPairs)
	for start := 0; start < len(order); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := start + 1
		for end < len(order) && end-start < batchPairs && isWithinBudget(end-start+1, len(pairs[order[end]].IDs), limit, batchPairs) {
			end++
		}
		batch = batch[:0]
		for _, index := range order[start:end] {
			batch = append(batch, pairs[index])
		}
		scores, err := pass(ctx, batch)
		if err != nil {
			return nil, err
		}
		if len(scores) != len(batch) {
			return nil, fmt.Errorf("got %d scores for %d pairs", len(scores), len(batch))
		}
		for i, index := range order[start:end] {
			out[index] = scores[i]
		}
		start = end
	}
	return out, nil
}

func (s *Scorer) forward(ctx context.Context, batch []tokenizer.Encoding) ([]float32, error) {
	rows, seq, padded := padBatch(s.paddings, batch, s.pairs.pad, s.pairs.maxTokens)
	defer s.paddings.Put(padded)

	session, err := s.sessions.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer s.sessions.release(session)

	shape := []int64{int64(rows), int64(seq)}
	in := map[string]*ort.Value{}
	for _, one := range []struct {
		name string
		flat []int64
	}{
		{inputIDs, padded.ids},
		{attentionMask, padded.mask},
		{tokenTypeIDs, padded.types},
	} {
		if one.name == tokenTypeIDs && !s.isTyped {
			continue
		}
		value, err := ort.NewTensor(shape, one.flat)
		if err != nil {
			return nil, err
		}
		defer value.Destroy()
		in[one.name] = value
	}

	reply, err := session.Run(in)
	runtime.KeepAlive(padded)
	if err != nil {
		return nil, err
	}
	for _, v := range reply {
		defer v.Destroy()
	}
	target, ok := reply[s.output]
	if !ok {
		return nil, fmt.Errorf("%s answered without %s", s.name, s.output)
	}
	outShape, err := target.GetShape()
	if err != nil {
		return nil, err
	}
	logits, err := ort.GetTensorData[float32](target)
	if err != nil {
		return nil, err
	}
	return readScores(logits, outShape)
}

func readScores(logits []float32, shape []int64) ([]float32, error) {
	if len(shape) != 2 || shape[1] != 1 {
		return nil, fmt.Errorf("the model answers logits of shape %v, expected one logit per pair", shape)
	}
	if int64(len(logits)) != shape[0] {
		return nil, fmt.Errorf("the model answers %d logits for %d pairs", len(logits), shape[0])
	}
	scores := make([]float32, len(logits))
	for i, logit := range logits {
		scores[i] = profile.GetScore(logit)
	}
	return scores, nil
}
