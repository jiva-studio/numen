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
type attempt func(ctx context.Context, provider connectionport.Provider, call modelcall.ProviderCall) ([]byte, error)

// asideRow is a connection set aside in the call being carried out.
type asideRow struct {
	id           string
	name         string
	revision     int
	reason       connection.FailureReason
	isDiscovered bool
}

type rowOutcome struct {
	value   []byte
	name    string
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

func carry(ctx context.Context, c *Calls, ask asking, do attempt) ([]byte, error) {
	var none []byte
	rows, err := c.rows(ask.callerPluginID, ask.operation)
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
		outcome := tryRow(ctx, c, row, ask, do)
		if outcome.err != nil {
			return none, outcome.err
		}
		if outcome.isDone {
			c.reportSetAside(setAside, outcome.name)
			return outcome.value, nil
		}
		failures = append(failures, *outcome.failure)
		if outcome.aside != nil {
			setAside = append(setAside, *outcome.aside)
		}
	}
	return none, &connection.AllFailedError{Failures: failures}
}

// callServices is what a row call takes from Calls: a slot of its connection, the provider and the secrets.
type callServices interface {
	acquire(ctx context.Context, connectionID string, capacity int, priority modelcall.Priority) (func(), error)
	openProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error)
	readConnectionSecrets(ctx context.Context, resolved connection.Resolved) (map[string]string, error)
}

func (c *Calls) acquire(ctx context.Context, connectionID string, capacity int, priority modelcall.Priority) (func(), error) {
	return c.queues.acquire(ctx, connectionID, capacity, priority)
}

func (c *Calls) openProvider(ctx context.Context, pluginID string) (connectionport.Provider, func(), error) {
	return c.source.OpenProvider(ctx, pluginID)
}

func (c *Calls) readConnectionSecrets(ctx context.Context, resolved connection.Resolved) (map[string]string, error) {
	return c.readSecrets(ctx, resolved)
}

// rowCall is one row being called: the connection as resolved, the secrets once they are read, and the
// operation to carry out.
type rowCall struct {
	calls    callServices
	resolved connection.Resolved
	row      connection.Row
	do       attempt
	priority modelcall.Priority
	secrets  map[string]string
}

func tryRow(ctx context.Context, c *Calls, row connection.Row, ask asking, do attempt) rowOutcome {
	outcome, isStale := tryResolvedRow(ctx, c, row, ask, do)
	if isStale {
		outcome, _ = tryResolvedRow(ctx, c, row, ask, do)
	}
	return outcome
}

func tryResolvedRow(ctx context.Context, c *Calls, row connection.Row, ask asking, do attempt) (rowOutcome, bool) {
	resolved, skip, err := c.checkRow(row, ask.operation)
	if err != nil {
		return rowOutcome{err: err}, false
	}
	if skip != nil {
		return rowOutcome{failure: skip}, false
	}
	call := &rowCall{calls: c, resolved: resolved, row: row, do: do, priority: ask.priority}
	for tries := 1; ; tries++ {
		value, err := call.callOnce(ctx)
		if err == nil {
			return rowOutcome{value: value, name: resolved.Name, isDone: true}, false
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rowOutcome{err: ctxErr}, false
		}
		judged := readVerdict(resolved, err)
		if judged.isCooling {
			c.cooldown.start(connectionVersion{id: resolved.ID, revision: resolved.Revision}, judged.reason, judged.retryAfter, c.clock.Now())
		}
		if judged.isSkipped || judged.err != nil {
			return rowOutcome{failure: judged.skip, err: judged.err}, judged.isStale
		}
		if tries == 1 && isRetried(judged.reason) {
			continue
		}
		c.cooldown.start(connectionVersion{id: resolved.ID, revision: resolved.Revision}, judged.reason, judged.retryAfter, c.clock.Now())
		return rowOutcome{
			failure: &connection.Failure{ConnectionName: resolved.Name, Reason: judged.reason.String()},
			aside:   &asideRow{id: resolved.ID, name: resolved.Name, revision: resolved.Revision, reason: judged.reason, isDiscovered: row.IsDiscovered},
		}, false
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
		return resolved, newSkip(resolved.Name, "the type does not serve "+operation), nil
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
	isStale    bool
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
		return verdict{isSkipped: true, skip: newSkip(resolved.Name, connection.FailureReasonUnavailable.String())}
	case errors.Is(err, connection.ErrSecretMissing):
		return verdict{isSkipped: true, skip: newSkip(resolved.Name, "key missing")}
	case errors.Is(err, connection.ErrKeyStore):
		return verdict{isSkipped: true, skip: newSkip(resolved.Name, "key store failed")}
	case errors.Is(err, connection.ErrStaleRevision):
		return verdict{isSkipped: true, isStale: true, skip: newSkip(resolved.Name, "changed during the call")}
	case errors.As(err, &stopError):
		return verdict{err: &connection.StoppedError{ConnectionName: resolved.Name, Reason: stopError.Reason.String()}}
	case errors.As(err, &tryNext):
		return verdict{reason: tryNext.Reason, retryAfter: tryNext.RetryAfter}
	}
	return verdict{reason: connection.FailureReasonUnavailable}
}

func isRetried(reason connection.FailureReason) bool {
	return reason == connection.FailureReasonUnavailable
}

func (r *rowCall) callOnce(ctx context.Context) ([]byte, error) {
	var none []byte
	freeSlot, err := r.calls.acquire(ctx, r.resolved.ID, r.resolved.ParallelRequests, r.priority)
	if err != nil {
		return none, err
	}
	defer freeSlot()
	provider, release, err := r.calls.openProvider(ctx, r.resolved.PluginID)
	if err != nil {
		return none, &openError{err: err}
	}
	defer release()
	if r.secrets == nil {
		secrets, err := r.calls.readConnectionSecrets(ctx, r.resolved)
		if err != nil {
			return none, err
		}
		r.secrets = secrets
	}
	return r.do(ctx, provider, r.resolved.NewCall(r.secrets, r.row.Options))
}

func (c *Calls) reportSetAside(rows []asideRow, nextName string) {
	for _, one := range rows {
		if one.isDiscovered {
			continue
		}
		if c.cooldown.claim(notice{version: connectionVersion{id: one.id, revision: one.revision}, reason: one.reason}) {
			c.notices(connection.SetAside{ConnectionID: one.id, ConnectionName: one.name, Reason: one.reason, NextName: nextName})
		}
	}
}
