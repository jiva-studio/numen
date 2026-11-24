package syncvault

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	keyUnreachable = "server-unreachable"
	keyFailed      = "sync-failed"
)

// Warner tells what the subscription means for the changes.
type Warner interface {
	Execute(ctx context.Context) error
}

// Pass is one pass over one vault.
type Pass interface {
	Execute(ctx context.Context, id domain.VaultID) error
}

// RunDeps are what the loop needs. Interval is read before each wait. Changed names a vault whose files changed.
type RunDeps struct {
	Pass     Pass
	Host     ports.Host
	State    ports.Links
	Health   ports.Health
	Clock    ports.Clock
	Warn     Warner
	Interval func() time.Duration
	Changed  <-chan domain.VaultID
	// Describe is the plain text of an error for the person; Report writes the failure of a pass to the log.
	Describe func(err error) string
	Report   func(ctx context.Context, vault domain.VaultID, err error)
}

// RunSync passes over every connected vault at a fixed interval, and at once over a vault whose files changed.
type RunSync struct{ deps RunDeps }

// NewRunSync returns the use case over the dependencies.
func NewRunSync(deps RunDeps) *RunSync { return &RunSync{deps: deps} }

// Execute runs until ctx ends and returns its error. A server that is away raises a warning and the loop waits the
// same interval again.
func (u *RunSync) Execute(ctx context.Context) error {
	err := u.runLoop(ctx)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}

func (u *RunSync) runLoop(ctx context.Context) error {
	if err := u.passAll(ctx); err != nil {
		return err
	}
	next := u.deps.Clock.Now().Add(u.deps.Interval())
	for {
		id, isChange, err := u.wait(ctx, next)
		if err != nil {
			return err
		}
		if isChange {
			err = u.passOne(ctx, id)
		} else {
			err = u.passAll(ctx)
			next = u.deps.Clock.Now().Add(u.deps.Interval())
		}
		if err != nil {
			return err
		}
	}
}

func (u *RunSync) wait(ctx context.Context, next time.Time) (domain.VaultID, bool, error) {
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	ended := make(chan error, 1)
	go func() { ended <- u.deps.Clock.Wait(waitCtx, max(0, next.Sub(u.deps.Clock.Now()))) }()
	select {
	case err := <-ended:
		return "", false, err
	case id := <-u.deps.Changed:
		cancel()
		if err := <-ended; err != nil && !errors.Is(err, context.Canceled) {
			return "", false, err
		}
		return id, true, ctx.Err()
	}
}

func (u *RunSync) passAll(ctx context.Context) error {
	vaults, err := u.deps.Host.GetVaults(ctx)
	if err != nil {
		return u.finish(ctx, &failure{vault: domain.Vault{}, err: err}, false)
	}
	var failed []failure
	isConnected := false
	for _, vault := range vaults {
		isPassed, err := u.pass(ctx, vault.ID)
		isConnected = isConnected || isPassed
		if err != nil {
			failed = append(failed, failure{vault: vault, err: err})
		}
	}
	return u.finish(ctx, u.pick(ctx, failed), isConnected)
}

func (u *RunSync) passOne(ctx context.Context, id domain.VaultID) error {
	isConnected, err := u.pass(ctx, id)
	if err == nil {
		return u.finish(ctx, nil, isConnected)
	}
	named := domain.Vault{ID: id, Name: "", Path: "", ManagedBy: ""}
	if vaults, listErr := u.deps.Host.GetVaults(ctx); listErr == nil {
		if at := slices.IndexFunc(vaults, func(vault domain.Vault) bool { return vault.ID == id }); at >= 0 {
			named = vaults[at]
		}
	}
	return u.finish(ctx, u.pick(ctx, []failure{{vault: named, err: err}}), isConnected)
}

// pick logs every failure of the pass and returns the one the health item tells of: a server that is away first.
func (u *RunSync) pick(ctx context.Context, failed []failure) *failure {
	var picked *failure
	for i := range failed {
		u.deps.Report(ctx, failed[i].vault.ID, failed[i].err)
		if picked == nil || (isUnreachable(failed[i].err) && !isUnreachable(picked.err)) {
			picked = &failed[i]
		}
	}
	return picked
}

func isUnreachable(err error) bool {
	return errors.Is(err, syncprotocol.ErrUnreachable) || errors.Is(err, sessiondomain.ErrUnreachable)
}

func (u *RunSync) pass(ctx context.Context, id domain.VaultID) (bool, error) {
	link, err := u.deps.State.ReadLink(ctx, id)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return false, nil
	}
	if err == nil && link.Kind == vaultlink.KindReceived {
		if err := u.deps.Pass.Execute(ctx, id); !errors.Is(err, share.ErrEnded) {
			return false, err
		}
		return false, nil
	}
	err = u.deps.Pass.Execute(ctx, id)
	if errors.Is(err, vaultlink.ErrNotConnected) {
		return false, nil
	}
	return true, err
}

func (u *RunSync) finish(ctx context.Context, failed *failure, isConnected bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if failed == nil && isConnected {
		if err := u.deps.Warn.Execute(ctx); err != nil {
			u.deps.Report(ctx, "", err)
			failed = &failure{vault: domain.Vault{}, err: err}
		}
	}
	if failed == nil {
		return u.clear(ctx)
	}
	if isUnreachable(failed.err) {
		return u.raise(ctx, health.Item{Key: keyUnreachable, Text: "Could not reach the sync server", Detail: "", Action: nil, Severity: health.Warning})
	}
	text := textFailed
	if name := failed.vault.Name; name != "" {
		text = "Could not sync the vault «" + truncate(name, maxName) + "»"
	}
	return u.raise(ctx, health.Item{Key: keyFailed, Text: text, Detail: truncate(u.deps.Describe(failed.err), maxDetail), Action: nil, Severity: health.Warning})
}

func truncate(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}

func (u *RunSync) raise(ctx context.Context, item health.Item) error {
	return u.deps.Health.Report(ctx, item)
}

func (u *RunSync) clear(ctx context.Context) error {
	for _, key := range []string{keyUnreachable, keyFailed} {
		if err := u.deps.Health.Clear(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

const (
	maxDetail  = 500
	maxName    = 60
	textFailed = "Could not sync a vault"
)

// failure is the error of the pass over one vault. A vault that is not known has an empty ID.
type failure struct {
	vault domain.Vault
	err   error
}
