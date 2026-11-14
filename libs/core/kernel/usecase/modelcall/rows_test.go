package modelcall_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall/modelcalltest"
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

func TestAnExclusiveRowThatFailsEndsTheCallWithItsFailure(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonRateLimited, 0))

	_, err := s.completePreferring(t, &modelcalldomain.PreferredRow{ConnectionID: s.two.ID, IsExclusive: true})

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("err = %v, want *AllFailedError", err)
	}
	want := []connection.Failure{{ConnectionName: "Second", Reason: "rate limited", Code: connection.FailureReasonRateLimited}}
	if !slices.Equal(failed.Failures, want) {
		t.Errorf("failures = %v, want %v", failed.Failures, want)
	}
	if len(s.world.Provider.CallsTo(addressOne)) != 0 {
		t.Error("a row other than the exclusive one was asked")
	}
}

func TestAnExclusiveRowIsAskedWithItsOptions(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)

	text, err := s.completePreferring(t, &modelcalldomain.PreferredRow{
		ConnectionID: s.two.ID, Options: map[string]string{"model": "model-three"}, IsExclusive: true,
	})

	if err != nil || text != "answer from "+addressTwo {
		t.Fatalf("Complete = %q, %v, want the answer of the second row", text, err)
	}
	if got := s.world.Provider.CallsTo(addressTwo)[0].Call.Options["model"]; got != "model-three" {
		t.Errorf("model = %q, want model-three", got)
	}
}

func TestAnExclusiveConnectionOutsideTheRowsGetsNoConnection(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)
	other := s.world.AddServer(t, "Third", "https://three.test/v1", "sk-three-0123456789")

	_, err := s.completePreferring(t, &modelcalldomain.PreferredRow{ConnectionID: other.ID, IsExclusive: true})

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Fatalf("err = %v, want ErrNoConnection", err)
	}
	if len(s.world.Provider.CallsTo(addressOne)) != 0 || len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("a row was asked for a connection that is not among the rows")
	}
}

func TestTheRowsAreListedWithTheirConnectionAndTheRowFieldsOfItsType(t *testing.T) {
	s := newSetup(t, modelcalltest.CompleteOperation)

	rows, err := newRows(s.world, &listedOptions{}).ListRows(consumerID, modelcalltest.CompleteOperation)

	if err != nil || len(rows) != 2 {
		t.Fatalf("ListRows = %+v, %v", rows, err)
	}
	first := rows[0]
	if first.ConnectionID != s.one.ID || first.ConnectionName != "First" || first.Options["model"] != "model-one" {
		t.Errorf("first row = %+v", first)
	}
	want := []modelcalldomain.RowField{{Key: "model", Title: "Model", OptionsFrom: modelcalltest.ModelsOperation, IsRequired: true}}
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

	rows, err := newRows(s.world, &listedOptions{}).ListRows(consumerID, modelcalltest.ChatOperation)

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

func newRows(w *modelcalltest.World, asked *listedOptions) *modelcall.Rows {
	return modelcall.NewRows(w.Resolve, w.Rows.ListRows,
		func(_ context.Context, connectionID, field string, options map[string]string, operations []string) ([]modelcalldomain.Option, error) {
			*asked = listedOptions{connectionID, field, options, operations}
			return []modelcalldomain.Option{{Value: "model-one", Title: "Model one"}}, nil
		},
	)
}

func TestTheValuesOfARowFieldAreListedForAConnectionAmongTheRows(t *testing.T) {
	s := newSetup(t, modelcalltest.ChatOperation)
	var asked listedOptions
	listing := newRows(s.world, &asked)

	options, err := listing.ListRowOptions(t.Context(), consumerID, modelcalltest.ChatOperation, s.two.ID, "reasoning", map[string]string{"model": "model-two"})

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
	listing := newRows(s.world, &asked)

	_, err := listing.ListRowOptions(t.Context(), consumerID, modelcalltest.ChatOperation, other.ID, "model", nil)

	if !errors.Is(err, connection.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
	if asked.connectionID != "" {
		t.Error("the provider was asked for a connection outside the rows")
	}
}
