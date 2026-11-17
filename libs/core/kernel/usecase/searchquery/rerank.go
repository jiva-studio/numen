package searchquery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/operationresult"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// RerankRows lists the rows of the Reranking setting of the core in the order of use, each with the name of
// its connection and its options.
type RerankRows interface {
	ListRows(owner, operation string) ([]modelcall.Row, error)
}

// RerankCodec writes the request of llm.rerank and reads its result. A request that cannot be written and a
// result that cannot be read are errors.
type RerankCodec interface {
	EncodeRequest(query, instruction string, documents []string) ([]byte, error)
	DecodeResult(payload []byte) ([]float64, error)
}

// reranker is what a rerank stage calls: the rows of the setting, the connections they name, the calls of
// the operation and the stored scores.
type reranker struct {
	calls   connectionport.ModelCalls
	codec   RerankCodec
	rows    RerankRows
	resolve connectionport.ResolveConnection
	results OperationResults
	clock   machineport.Clock
	limits  searchquery.StageLimits
}

// rerankInput is what one rerank stage sends: the query, the instruction and the texts of the units, cut.
type rerankInput struct {
	query, instruction string
	texts              []string
	priority           modelcall.Priority
}

func (rr reranker) score(ctx context.Context, in rerankInput) ([]float64, searchquery.RerankReport, error) {
	rows, err := rr.rows.ListRows(settingsdomain.AppOwner, operationresult.OperationRerank)
	if err != nil {
		return nil, searchquery.RerankReport{}, err
	}
	if len(rows) == 0 {
		return nil, searchquery.RerankReport{}, searchquery.ErrNoReranker
	}
	var failures []searchquery.RerankFailure
	for _, row := range rows {
		scores, report, err := rr.scoreWith(ctx, row, in)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, searchquery.RerankReport{}, ctxErr
		}
		if err == nil {
			return scores, report, nil
		}
		failures = append(failures, searchquery.RerankFailure{ConnectionName: row.ConnectionName, Reason: getFailureReason(err)})
	}
	return nil, searchquery.RerankReport{}, &searchquery.RerankError{Failures: failures}
}

const rerankBatchSize = 32

var errUnavailable = errors.New("not available")

func getFailureReason(err error) string {
	if failed, ok := errors.AsType[*connection.AllFailedError](err); ok {
		reasons := make([]string, len(failed.Failures))
		for i, one := range failed.Failures {
			reasons[i] = one.Reason
		}
		return strings.Join(reasons, "; ")
	}
	if stopped, ok := errors.AsType[*connection.StoppedError](err); ok {
		return stopped.Reason
	}
	return err.Error()
}

func (rr reranker) scoreWith(ctx context.Context, row modelcall.Row, in rerankInput) ([]float64, searchquery.RerankReport, error) {
	resolved, err := rr.resolve(row.ConnectionID)
	if err != nil {
		return nil, searchquery.RerankReport{}, errUnavailable
	}
	recipe := operationresult.GetRerankRecipeHash(operationresult.RerankRecipe{
		Fingerprint: resolved.Fingerprint, ProviderVersion: resolved.PluginVersion, Options: row.Options,
		MaxRerankedBytes: rr.limits.MaxRerankedBytes,
	})
	hashes := make([]string, len(in.texts))
	for i, text := range in.texts {
		hashes[i] = operationresult.GetRerankInputHash(in.query, in.instruction, text)
	}
	stored, err := rr.results.FindResults(ctx, operationresult.OperationRerank, recipe, hashes, rr.clock.Now())
	if err != nil {
		return nil, searchquery.RerankReport{}, err
	}
	scores, missing := getStoredScores(hashes, stored)
	report := searchquery.RerankReport{
		ConnectionID: row.ConnectionID, ConnectionName: row.ConnectionName, Options: row.Options,
		SentCount: len(missing), ReusedCount: len(hashes) - len(missing),
	}
	if len(missing) == 0 {
		return scores, report, nil
	}
	if err := rr.callRow(ctx, row, in, missing, scores); err != nil {
		return nil, searchquery.RerankReport{}, err
	}
	outputs := make(map[string][]byte, len(missing))
	for _, i := range missing {
		outputs[hashes[i]] = operationresult.EncodeScore(scores[i])
	}
	if err := rr.results.SaveResults(ctx, operationresult.OperationRerank, recipe, outputs, rr.clock.Now()); err != nil {
		return nil, searchquery.RerankReport{}, err
	}
	return scores, report, nil
}

func getStoredScores(hashes []string, stored map[string][]byte) ([]float64, []int) {
	scores := make([]float64, len(hashes))
	var missing []int
	for i, hash := range hashes {
		output, found := stored[hash]
		score, err := operationresult.DecodeScore(output)
		if !found || err != nil {
			missing = append(missing, i)
			continue
		}
		scores[i] = score
	}
	return scores, missing
}

func (rr reranker) callRow(ctx context.Context, row modelcall.Row, in rerankInput, missing []int, scores []float64) error {
	for start := 0; start < len(missing); start += rerankBatchSize {
		if err := rr.callBatch(ctx, row, in, missing[start:min(start+rerankBatchSize, len(missing))], scores); err != nil {
			return err
		}
	}
	return nil
}

func (rr reranker) callBatch(ctx context.Context, row modelcall.Row, in rerankInput, missing []int, scores []float64) error {
	documents := make([]string, len(missing))
	for j, i := range missing {
		documents[j] = in.texts[i]
	}
	payload, err := rr.codec.EncodeRequest(in.query, in.instruction, documents)
	if err != nil {
		return err
	}
	done, err := rr.calls.Call(ctx, settingsdomain.AppOwner, modelcall.Request{
		Operation: operationresult.OperationRerank, Payload: payload,
		Priority:  in.priority,
		Preferred: &modelcall.PreferredRow{ConnectionID: row.ConnectionID, Options: row.Options, IsExclusive: true},
	}, func([]byte) error { return nil })
	if err != nil {
		return err
	}
	got, err := rr.codec.DecodeResult(done)
	if err != nil {
		return err
	}
	if len(got) != len(documents) {
		return fmt.Errorf("the service answered %d scores for %d passages", len(got), len(documents))
	}
	for j, i := range missing {
		scores[i] = got[j]
	}
	return nil
}
