package cli_test

import (
	"math"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	connectionusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/connection/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery/searchquerytest"
)

const rerankAddress = "https://rerank.test/v1"

func answerEachPassage(payload []byte) ([]byte, error) {
	var packed []byte
	for len(payload) > 0 {
		number, _, n := protowire.ConsumeTag(payload)
		payload = payload[n:]
		_, m := protowire.ConsumeBytes(payload)
		payload = payload[m:]
		if number == 3 {
			packed = protowire.AppendFixed32(packed, math.Float32bits(0.5))
		}
	}
	return protowire.AppendBytes(protowire.AppendTag(nil, 1, protowire.BytesType), packed), nil
}

func newRerankingQuery(t *testing.T) stubSession {
	t.Helper()
	w := modelcalltest.New()
	reranker := w.AddReranker(t, "Harbor reranker", rerankAddress)
	w.Rows.Set(settingsdomain.AppOwner, modelcalltest.RerankOperation, connection.Row{ConnectionID: reranker.ID, Options: map[string]string{"model": "rerank-small"}})
	w.Provider.Answer(rerankAddress, answerEachPassage)
	return newQuerySession(t, func(db *index.DB, deps *searchquerytest.Deps) {
		deps.Stages.Calls = connectionusecase.NewCalls(
			connectionusecase.Connections{Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows.ListRows},
			w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside,
		)
		deps.Stages.Rows = connectionusecase.NewRows(w.Resolve, w.Rows.ListRows, nil)
		deps.Stages.Resolve, deps.Stages.Results, deps.Clock = w.Resolve, db.GetOperationResults(100), w.Clock
	})
}

func TestQueryWithARerankPrintsTheResultsAndWhichServiceAnswered(t *testing.T) {
	out, stderr, err := runWithStderr(t, newRerankingQuery(t), "query", "shelf", `words("tide").rerank("tide")`)
	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	if !strings.Contains(out, "notes/Moon.md") || !strings.Contains(out, "notes/Harbor.md") {
		t.Errorf("printed %q, want both tide notes", out)
	}
	if !strings.Contains(stderr, "rerank: Harbor reranker (model=rerank-small) sent 2, reused 0") {
		t.Errorf("stderr is %q, want the rerank report", stderr)
	}
}

func TestOperationResultsClearPrintsHowManyItRemoved(t *testing.T) {
	session := newRerankingQuery(t)
	if _, stderr, err := runWithStderr(t, session, "query", "shelf", `words("tide").rerank("tide")`); err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}

	out, stderr, err := runWithStderr(t, session, "operation-results", "clear")

	if err != nil {
		t.Fatalf("%v: %s", err, stderr)
	}
	if out != "removed 2 stored results\n" {
		t.Errorf("printed %q", out)
	}
	if _, _, err := runWithStderr(t, session, "operation-results"); err == nil || !strings.Contains(err.Error(), "usage: numen-cli operation-results clear") {
		t.Errorf("operation-results alone returned %v, want the usage", err)
	}
}
