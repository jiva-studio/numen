package modelcall

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/usecase/connection"
)

// attempt is what one operation does with a provider once the call to it is made.
type attempt[R, T any] func(ctx context.Context, provider R, call connectionport.ProviderCall, row connectionport.Row) (T, error)

// asideRow is a connection set aside in the call being carried out.
type asideRow struct {
	id       string
	name     string
	revision int
	reason   connectionport.FailureReason
}

type rowOutcome[T any] struct {
	value   T
	failure *connectionport.Failure
	aside   *asideRow
	err     error
	isDone  bool
}

// openError is a provider that could not be started.
type openError struct{ err error }

func (e *openError) Error() string { return e.err.Error() }

func carry[R, T any](
	ctx context.Context, c *Calls, callerPluginID, operation string, pick func(connectionport.Provider) R, do attempt[R, T],
) (T, error) {
	var none T
	rows, err := c.rows.ListRows(callerPluginID, operation)
	if err != nil {
		return none, fmt.Errorf("read the rows of %s: %w", callerPluginID, err)
	}
	if len(rows) == 0 {
		return none, connectionport.ErrNoConnection
	}
	var failures []connectionport.Failure
	var setAside []asideRow
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return none, err
		}
		outcome := tryRow(ctx, c, row, operation, pick, do)
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
	return none, &connectionport.AllFailedError{Failures: failures}
}

func tryRow[R, T any](
	ctx context.Context, c *Calls, row connectionport.Row, operation string, pick func(connectionport.Provider) R, do attempt[R, T],
) rowOutcome[T] {
	resolved, err := c.connections.Resolve.Execute(row.ConnectionID)
	switch {
	case errors.Is(err, connection.ErrNotFound):
		return skipped[T](row.ConnectionID, "connection missing")
	case errors.Is(err, connection.ErrTypeUnavailable):
		return skipped[T](row.ConnectionID, "type unavailable")
	case err != nil:
		return rowOutcome[T]{err: err}
	}
	if !resolved.Declaration.HasOperation(operation) {
		return skipped[T](resolved.Name, "this type cannot "+operation)
	}
	version := connectionVersion{id: resolved.ID, revision: resolved.Revision}
	if cooling, isCooling := c.cooldown.get(version, c.clock.Now()); isCooling {
		return skipped[T](resolved.Name, cooling.reason.String())
	}
	var secrets map[string]string
	for tries := 1; ; tries++ {
		value, err := callOnce(ctx, c, resolved, row, &secrets, pick, do)
		if err == nil {
			return rowOutcome[T]{value: value, isDone: true}
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return rowOutcome[T]{err: ctxErr}
		}
		var unavailable *openError
		var stopped *connectionport.StopError
		switch {
		case errors.As(err, &unavailable):
			return skipped[T](resolved.Name, connectionport.FailureReasonUnreachable.String())
		case errors.Is(err, connection.ErrSecretMissing):
			return skipped[T](resolved.Name, "key missing")
		case errors.Is(err, connection.ErrKeyStore):
			return skipped[T](resolved.Name, "key store failed")
		case errors.Is(err, connection.ErrStaleRevision):
			return rowOutcome[T]{err: fmt.Errorf("call %s: %w", resolved.Name, err)}
		case errors.As(err, &stopped):
			return rowOutcome[T]{err: &connectionport.RejectedError{ConnectionName: resolved.Name, Reason: stopped.Reason.String()}}
		}
		reason, retryAfter := readTryNext(err)
		if tries == 1 && isRetried(reason) {
			continue
		}
		c.cooldown.start(version, reason, retryAfter, c.clock.Now())
		return rowOutcome[T]{
			failure: &connectionport.Failure{ConnectionName: resolved.Name, Reason: reason.String()},
			aside:   &asideRow{id: resolved.ID, name: resolved.Name, revision: resolved.Revision, reason: reason},
		}
	}
}

func skipped[T any](name, reason string) rowOutcome[T] {
	return rowOutcome[T]{failure: &connectionport.Failure{ConnectionName: name, Reason: reason}}
}

// readTryNext returns the reason and the wait of a failure that is not a stop. An error of no failure
// kind is a provider that could not be reached.
func readTryNext(err error) (connectionport.FailureReason, time.Duration) {
	var tryNext *connectionport.TryNextError
	if errors.As(err, &tryNext) {
		return tryNext.Reason, tryNext.RetryAfter
	}
	return connectionport.FailureReasonUnreachable, 0
}

func isRetried(reason connectionport.FailureReason) bool {
	return reason == connectionport.FailureReasonTimeout || reason == connectionport.FailureReasonUnreachable
}

// callOnce opens the provider of the connection, reads the secrets on the first call and calls it. The
// provider is released when the call returns.
func callOnce[R, T any](
	ctx context.Context, c *Calls, resolved connection.Resolved, row connectionport.Row, secrets *map[string]string,
	pick func(connectionport.Provider) R, do attempt[R, T],
) (T, error) {
	var none T
	provider, release, err := c.source.OpenProvider(ctx, resolved.PluginID)
	if err != nil {
		return none, &openError{err: err}
	}
	defer release()
	if *secrets == nil {
		read, err := c.connections.ReadSecrets.Execute(ctx, resolved)
		if err != nil {
			return none, err
		}
		*secrets = read
	}
	return do(ctx, pick(provider), resolved.NewCall(*secrets, row.Model, row.Reasoning), row)
}

func (c *Calls) reportSetAside(rows []asideRow) {
	for _, one := range rows {
		if c.told.claim(notice{version: connectionVersion{id: one.id, revision: one.revision}, reason: one.reason}) {
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
