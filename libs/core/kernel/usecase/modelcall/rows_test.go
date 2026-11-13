package modelcall_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

func (s *setup) completePreferring(t *testing.T, preferred *modelcalldomain.PreferredRow) (string, error) {
	t.Helper()
	done, err := s.calls.Call(t.Context(), consumerID, modelcalldomain.Request{
		Operation: modelcalltest.CompleteOperation, Payload: []byte("fix this"), Preferred: preferred,
	}, func([]byte) error { return nil })
	return string(done), err
}

func TestAPreferredRowIsAskedFirstWithItsOptions(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)

	text, err := s.completePreferring(t, &modelcalldomain.PreferredRow{
		ConnectionID: s.two.ID, Options: map[string]string{"model": "model-three", "reasoning": "high"},
	})

	if err != nil || text != "answer from "+addressTwo {
		t.Fatalf("Complete = %q, %v, want the answer of the second row", text, err)
	}
	if len(s.world.Provider.CallsTo(addressOne)) != 0 {
		t.Error("the first row was asked before the preferred one")
	}
	got := s.world.Provider.CallsTo(addressTwo)[0].Call.Options
	if got["model"] != "model-three" || got["reasoning"] != "high" {
		t.Errorf("options = %v, want the preferred ones", got)
	}
}

func TestTheRowsAfterAPreferredRowThatFailsAreAskedInOrder(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.Fail(addressTwo, tryNext(connection.FailureReasonRateLimited, 0))

	text, err := s.completePreferring(t, &modelcalldomain.PreferredRow{ConnectionID: s.two.ID})

	if err != nil || text != "answer from "+addressOne {
		t.Fatalf("Complete = %q, %v, want the answer of the first row", text, err)
	}
	if got := len(s.world.Provider.CallsTo(addressTwo)); got != 1 {
		t.Errorf("the preferred row was asked %d times, want once", got)
	}
}

func TestAPreferredConnectionOutsideTheRowsKeepsTheirOrder(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	other := s.world.AddServer(t, "Third", "https://three.test/v1", "sk-three-0123456789")

	text, err := s.completePreferring(t, &modelcalldomain.PreferredRow{ConnectionID: other.ID})

	if err != nil || text != "answer from "+addressOne {
		t.Fatalf("Complete = %q, %v, want the answer of the first row", text, err)
	}
	if len(s.world.Provider.CallsTo("https://three.test/v1")) != 0 {
		t.Error("a connection that is not among the rows was called")
	}
}

func TestTheRowsAreListedWithTheirConnectionAndTheRowFieldsOfItsType(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)

	rows, err := s.calls.ListRows(consumerID, modelcalltest.CompleteOperation)

	if err != nil || len(rows) != 2 {
		t.Fatalf("ListRows = %+v, %v", rows, err)
	}
	first := rows[0]
	if first.ConnectionID != s.one.ID || first.ConnectionName != "First" || first.Options["model"] != "model-one" {
		t.Errorf("first row = %+v", first)
	}
	want := []modelcalldomain.RowField{{Key: "model", Title: "Model", OptionsFrom: modelcalltest.ModelsOperation}}
	if !slices.Equal(first.Fields, want) {
		t.Errorf("fields = %+v, want %+v", first.Fields, want)
	}
	if rows[1].ConnectionName != "Second" {
		t.Errorf("second row = %+v", rows[1])
	}
}

func TestTheRowsListOmitsAConnectionThatCannotServeTheOperation(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	completer := s.world.Add(t, modelcalltest.CompleterTypeID, "Completer", "https://three.test/v1", "")
	s.world.Rows.Set(consumerID, modelcalltest.ChatOperation, modelcalltest.Row(completer, "m"), modelcalltest.Row(s.one, "m"))

	rows, err := s.calls.ListRows(consumerID, modelcalltest.ChatOperation)

	if err != nil || len(rows) != 1 || rows[0].ConnectionName != "First" {
		t.Fatalf("ListRows = %+v, %v, want the first connection alone", rows, err)
	}
}

type listedOptions struct {
	connectionID string
	field        string
	options      map[string]string
	operations   []string
}

func newListingCalls(w *modelcalltest.World, asked *listedOptions) *modelcall.Calls {
	return modelcall.NewCalls(modelcall.Connections{
		Resolve: w.Resolve, ReadSecrets: w.ReadSecrets, Rows: w.Rows.ListRows,
		ListOptions: func(_ context.Context, connectionID, field string, options map[string]string, operations []string) ([]modelcalldomain.Option, error) {
			*asked = listedOptions{connectionID, field, options, operations}
			return []modelcalldomain.Option{{Value: "model-one", Title: "Model one"}}, nil
		},
	}, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside)
}

func TestTheValuesOfARowFieldAreListedForAConnectionAmongTheRows(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	var asked listedOptions
	calls := newListingCalls(s.world, &asked)

	options, err := calls.ListRowOptions(t.Context(), consumerID, modelcalltest.ChatOperation, s.two.ID, "reasoning", map[string]string{"model": "model-two"})

	if err != nil || len(options) != 1 || options[0].Value != "model-one" {
		t.Fatalf("ListRowOptions = %+v, %v", options, err)
	}
	if asked.connectionID != s.two.ID || asked.field != "reasoning" || asked.options["model"] != "model-two" ||
		!slices.Equal(asked.operations, []string{modelcalltest.ChatOperation}) {
		t.Errorf("asked %+v", asked)
	}
}

func TestTheValuesOfARowFieldOfAConnectionOutsideTheRowsAreNotFound(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	other := s.world.AddServer(t, "Third", "https://three.test/v1", "sk-three-0123456789")
	var asked listedOptions
	calls := newListingCalls(s.world, &asked)

	_, err := calls.ListRowOptions(t.Context(), consumerID, modelcalltest.ChatOperation, other.ID, "model", nil)

	if !errors.Is(err, connection.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
	if asked.connectionID != "" {
		t.Error("the provider was asked for a connection outside the rows")
	}
}
