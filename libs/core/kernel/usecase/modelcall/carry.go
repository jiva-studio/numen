package modelcall

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// attempt is what one operation does with a provider once the call to it is made.
type attempt[R, T any] func(ctx context.Context, provider R, call modelcall.ProviderCall, row connection.Row) (T, error)

// asideRow is a connection set aside in the call being carried out.
type asideRow struct {
	id       string
	name     string
	revision int
	reason   connection.FailureReason
}

type rowOutcome[T any] struct {
	value   T
	failure *connection.Failure
	aside   *asideRow
	err     error
	isDone  bool
}

// openError is a provider that could not be started.
type openError struct{ err error }

func (e *openError) Error() string { return e.err.Error() }

// asking is what one call of a plugin asks for.
type asking struct {
	callerPluginID string
	operation      string
	priority       modelcall.Priority
}

func carry[R, T any](
	ctx context.Context, c *Calls, ask asking, pick func(connectionport.Provider) R, do attempt[R, T],
) (T, error) {
	var none T
	rows, err := c.rows.ListRows(ask.callerPluginID, ask.operation)
	if err != nil {
		return none, fmt.Errorf("read the rows of %s: %w", ask.callerPluginID, err)
	}
	if len(rows) == 0 {
		return none, connection.ErrNoConnection
	}
	var failures []connection.Failure
	var setAside []asideRow
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return none, err
		}
		outcome := tryRow(ctx, c, row, ask, pick, do)
		if outcome.err != nil {
			return none, outcome.err
		}
		if outcome.isDone {
			c.reportSetAside(setAside)
			return outcome.value, nil
		}
		failures = append(failures, *outcome.failure)
		if outcome.aside != nil {
			setAside = append(setAside, *outcome.aside)
		}
	}
	return none, &connection.AllFailedError{Failures: failures}
}

// rowCall is one row being called: the connection as resolved, the secrets once they are read, and the
// operation to carry out.
type rowCall[R, T any] struct {
	calls    *Calls
	resolved connection.Resolved
	row      connection.Row
	pick     func(connectionport.Provider) R
	do       attempt[R, T]
	priority modelcall.Priority
	secrets  map[string]string
}

func tryRow[R, T any](
	ctx context.Context, c *Calls, row connection.Row, ask asking, pick func(connectionport.Provider) R, do attempt[R, T],
) rowOutcome[T] {
	resolved, skip, err := c.checkRow(row, ask.operation)
	if err != nil {
		return rowOutcome[T]{err: err}
	}
	if skip != nil {
		return rowOutcome[T]{failure: skip}
	}
	call := &rowCall[R, T]{calls: c, resolved: resolved, row: row, pick: pick, do: do, priority: ask.priority}
	for tries := 1; ; tries++ {
		value, err := call.callOnce(ctx)
		if err == nil {
			return rowOutcome[T]{value: value, isDone: true}
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rowOutcome[T]{err: ctxErr}
		}
		judged := readVerdict(resolved, err)
		if judged.isCooling {
			c.cooldown.start(connectionVersion{id: resolved.ID, revision: resolved.Revision}, judged.reason, judged.retryAfter, c.clock.Now())
		}
		if judged.isSkipped || judged.err != nil {
			return rowOutcome[T]{failure: judged.skip, err: judged.err}
		}
		if tries == 1 && isRetried(judged.reason) {
			continue
		}
		c.cooldown.start(connectionVersion{id: resolved.ID, revision: resolved.Revision}, judged.reason, judged.retryAfter, c.clock.Now())
		return rowOutcome[T]{
			failure: &connection.Failure{ConnectionName: resolved.Name, Reason: judged.reason.String()},
			aside:   &asideRow{id: resolved.ID, name: resolved.Name, revision: resolved.Revision, reason: judged.reason},
		}
	}
}

func (c *Calls) checkRow(row connection.Row, operation string) (connection.Resolved, *connection.Failure, error) {
	resolved, err := c.resolve(row.ConnectionID)
	switch {
	case errors.Is(err, connection.ErrNotFound):
		return resolved, newSkip(row.ConnectionID, "connection missing"), nil
	case errors.Is(err, connection.ErrTypeUnavailable):
		return resolved, newSkip(row.ConnectionID, "type unavailable"), nil
	case err != nil:
		return resolved, nil, err
	}
	if !resolved.Declaration.HasOperation(operation) {
		return resolved, newSkip(resolved.Name, "this type cannot "+operation), nil
	}
	version := connectionVersion{id: resolved.ID, revision: resolved.Revision}
	if cooling, isCooling := c.cooldown.get(version, c.clock.Now()); isCooling {
		return resolved, newSkip(resolved.Name, cooling.reason.String()), nil
	}
	return resolved, nil, nil
}

func newSkip(name, reason string) *connection.Failure {
	return &connection.Failure{ConnectionName: name, Reason: reason}
}

// verdict is what a failed call means for its row.
type verdict struct {
	isSkipped  bool
	skip       *connection.Failure
	err        error
	reason     connection.FailureReason
	retryAfter time.Duration
	isCooling  bool
}

func readVerdict(resolved connection.Resolved, err error) verdict {
	var unavailable *openError
	var stopError *connection.StopError
	var tryNext *connection.TryNextError
	var consumer *consumerError
	var midStream *midStreamError
	switch {
	case errors.As(err, &consumer):
		return verdict{err: consumer.err}
	case errors.As(err, &midStream):
		return readMidStreamVerdict(resolved.Name, midStream.err)
	case errors.Is(err, errQueueTimeout):
		return verdict{isSkipped: true, skip: newSkip(resolved.Name, "busy")}
	case errors.As(err, &unavailable):
		return verdict{isSkipped: true, skip: newSkip(resolved.Name, connection.FailureReasonUnreachable.String())}
	case errors.Is(err, connection.ErrSecretMissing):
		return verdict{isSkipped: true, skip: newSkip(resolved.Name, "key missing")}
	case errors.Is(err, connection.ErrKeyStore):
		return verdict{isSkipped: true, skip: newSkip(resolved.Name, "key store failed")}
	case errors.Is(err, connection.ErrStaleRevision):
		return verdict{err: fmt.Errorf("call %s: %w", resolved.Name, err)}
	case errors.As(err, &stopError):
		return verdict{err: &connection.RejectedError{ConnectionName: resolved.Name, Reason: stopError.Reason.String()}}
	case errors.As(err, &tryNext):
		return verdict{reason: tryNext.Reason, retryAfter: tryNext.RetryAfter}
	}
	return verdict{reason: connection.FailureReasonUnreachable}
}

func isRetried(reason connection.FailureReason) bool {
	return reason == connection.FailureReasonTimeout || reason == connection.FailureReasonUnreachable
}

func (r *rowCall[R, T]) callOnce(ctx context.Context) (T, error) {
	var none T
	freeSlot, err := r.calls.queues.acquire(ctx, r.resolved.ID, r.resolved.ParallelRequests, r.priority)
	if err != nil {
		return none, err
	}
	defer freeSlot()
	provider, release, err := r.calls.source.OpenProvider(ctx, r.resolved.PluginID)
	if err != nil {
		return none, &openError{err: err}
	}
	defer release()
	if r.secrets == nil {
		secrets, err := r.calls.readSecrets(ctx, r.resolved)
		if err != nil {
			return none, err
		}
		r.secrets = secrets
	}
	return r.do(ctx, r.pick(provider), r.resolved.NewCall(r.secrets, r.row.Model, r.row.Reasoning), r.row)
}

func (c *Calls) reportSetAside(rows []asideRow) {
	for _, one := range rows {
		if c.noticeLog.claim(notice{version: connectionVersion{id: one.id, revision: one.revision}, reason: one.reason}) {
			c.notices.ReportSetAside(one.id, one.name, one.reason)
		}
	}
}

func pickCompletion(provider connectionport.Provider) connectionport.CompletionProvider {
	return provider
}

func pickEmbedding(provider connectionport.Provider) connectionport.EmbeddingProvider {
	return provider
}

func pickProfiles(provider connectionport.Provider) connectionport.ModelProfiles { return provider }
