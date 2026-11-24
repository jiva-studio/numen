// Package session keeps this device signed in to the sync server.
package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
)

const (
	retryWithoutIdentity = 5 * time.Second
	retryAfterFailure    = 30 * time.Second
	renewBeforeExpiry    = 60 * time.Second
	shortestRenewal      = 5 * time.Second
	hostCallTimeout      = 30 * time.Second
)

// Identity is the identity of this device, which the host keeps.
type Identity interface {
	// GetPublicIdentity returns session.ErrNoIdentity where the device holds none.
	GetPublicIdentity(ctx context.Context) (domain.Identity, error)
	// SignAsDevice signs the message with the key of the device.
	SignAsDevice(ctx context.Context, message []byte) ([]byte, error)
}

// Server is the sync server. Its errors are session.ErrUpdateRequired, session.ErrUnauthenticated and
// session.ErrUnreachable where they apply.
type Server interface {
	// GetChallenge asks for a challenge and returns it ready to sign, or an error for one it must not sign.
	GetChallenge(ctx context.Context, deviceSignKey []byte) (domain.Challenge, error)
	// SignIn sends the proof and returns the session.
	SignIn(ctx context.Context, proof domain.Proof) (domain.Session, error)
}

// Health is where the plugin says what does not work.
type Health interface {
	Report(ctx context.Context, item health.Item) error
	Clear(ctx context.Context, key string) error
}

// Clock reads the time and waits.
type Clock interface {
	Now() time.Time
	// Wait returns after d, or when ctx ends with its error.
	Wait(ctx context.Context, d time.Duration) error
}

// Chains is the core that takes the chain of the server which says this device was revoked.
type Chains interface {
	// AdoptChain gives the core a chain the server holds.
	AdoptChain(ctx context.Context, chain []byte) error
}

// Deps are what Keep signs in with. Chains is optional.
type Deps struct {
	Identity Identity
	Server   Server
	Health   Health
	Clock    Clock
	Chains   Chains
}

// concern is an item of health the plugin raises, and how long it waits before it tries again.
type concern struct {
	key, text string
	retry     time.Duration
}

func newNoIdentityConcern() concern {
	return concern{"no-identity", "No identity on this device", retryWithoutIdentity}
}

func newServerUnreachableConcern() concern {
	return concern{"server-unreachable", "Could not reach the sync server", retryAfterFailure}
}

func newUpdateRequiredConcern() concern {
	return concern{"update-required", "Update Numen", retryAfterFailure}
}

func newRevokedConcern() concern {
	return concern{"device-revoked", "This device was revoked", retryAfterFailure}
}

func newAccountDeletingConcern() concern {
	return concern{"account-deleting", "Your cloud data is being deleted", retryAfterFailure}
}

func newSignInFailedConcern() concern {
	return concern{"sign-in-failed", "Could not sign in", retryAfterFailure}
}

// Keep signs in with the identity of this device, signs in again before the session ends, and tries
// again at a fixed interval where it cannot.
type Keep struct {
	deps    Deps
	raised  string
	isClean bool

	mu         sync.Mutex
	token      string
	deleteAt   time.Time
	isDeleting bool
}

// NewKeep returns the use case over the dependencies.
func NewKeep(deps Deps) *Keep {
	return &Keep{deps: deps, raised: "", isClean: false, mu: sync.Mutex{}, token: "", deleteAt: time.Time{}, isDeleting: false}
}

// Execute runs until ctx ends and returns its error.
func (u *Keep) Execute(ctx context.Context) error {
	for {
		wait := u.signIn(ctx)
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := u.deps.Clock.Wait(ctx, wait); err != nil {
			return err
		}
	}
}

func (u *Keep) signIn(ctx context.Context) time.Duration {
	current, err := u.open(ctx, false)
	if err != nil {
		return u.raise(ctx, err)
	}
	u.clear(ctx)
	return max(current.ExpiresAt.Sub(u.deps.Clock.Now())-renewBeforeExpiry, shortestRenewal)
}

// Refresh signs in once more and returns why it could not.
func (u *Keep) Refresh(ctx context.Context) error {
	_, err := u.open(ctx, false)
	return err
}

// Restore signs in with the request to cancel the deletion of the account, and clears the warning of the deletion.
func (u *Keep) Restore(ctx context.Context) error {
	if _, err := u.open(ctx, true); err != nil {
		return err
	}
	u.clear(ctx)
	return nil
}

// GetDeletion returns when the server deletes the account for good, and false when the last sign-in found it not deleting.
func (u *Keep) GetDeletion() (time.Time, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.deleteAt, u.isDeleting
}

// SetDeletion records that the account is deleting, for the person who just asked for it.
func (u *Keep) SetDeletion(deleteAt time.Time) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.deleteAt, u.isDeleting, u.token = deleteAt, true, ""
}

// GetToken returns the token of the session, empty before the first sign-in.
func (u *Keep) GetToken() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.token
}

func (u *Keep) open(ctx context.Context, restore bool) (domain.Session, error) {
	identityCtx, cancelIdentity := context.WithTimeout(ctx, hostCallTimeout)
	public, err := u.deps.Identity.GetPublicIdentity(identityCtx)
	cancelIdentity()
	if err != nil {
		return domain.Session{}, err
	}
	challenge, err := u.deps.Server.GetChallenge(ctx, public.DeviceSignKey)
	if err != nil {
		return domain.Session{}, err
	}
	signCtx, cancelSign := context.WithTimeout(ctx, hostCallTimeout)
	signature, err := u.deps.Identity.SignAsDevice(signCtx, challenge.Message)
	cancelSign()
	if err != nil {
		return domain.Session{}, err
	}
	current, err := u.deps.Server.SignIn(ctx, domain.Proof{
		ChallengeID: challenge.ID, DeviceSignKey: public.DeviceSignKey, Signature: signature, Chain: public.Chain, RestoreAccount: restore,
	})
	u.mu.Lock()
	defer u.mu.Unlock()
	if deleting, isDeleting := errors.AsType[domain.AccountDeletingError](err); isDeleting {
		u.deleteAt, u.isDeleting, u.token = deleting.DeleteAt, true, ""
	}
	if err != nil {
		return domain.Session{}, err
	}
	u.deleteAt, u.isDeleting = time.Time{}, false
	u.token = current.Token
	return current, nil
}

func (u *Keep) raise(ctx context.Context, cause error) time.Duration {
	if ctx.Err() != nil {
		return 0
	}
	what := newSignInFailedConcern()
	revoked, isRevoked := errors.AsType[domain.DeviceRevokedError](cause)
	isDeleting := errors.Is(cause, domain.AccountDeletingError{DeleteAt: time.Time{}})
	switch {
	case isRevoked:
		what = newRevokedConcern()
		if u.adopt(ctx, revoked) != nil {
			what = newSignInFailedConcern()
		}
	case isDeleting:
		what = newAccountDeletingConcern()
	case errors.Is(cause, domain.ErrNoIdentity):
		what = newNoIdentityConcern()
	case errors.Is(cause, domain.ErrUpdateRequired):
		what = newUpdateRequiredConcern()
	case errors.Is(cause, domain.ErrUnreachable):
		what = newServerUnreachableConcern()
	}
	if u.raised == what.key {
		return what.retry
	}
	if u.clear(ctx); u.raised != "" {
		return what.retry
	}
	reportCtx, cancel := context.WithTimeout(ctx, hostCallTimeout)
	defer cancel()
	if u.deps.Health.Report(reportCtx, health.Item{Key: what.key, Severity: health.Warning, Text: what.text}) == nil {
		u.raised = what.key
	}
	return what.retry
}

func (u *Keep) clear(ctx context.Context) {
	if u.raised == "" {
		u.clearLeft(ctx)
		return
	}
	clearCtx, cancel := context.WithTimeout(ctx, hostCallTimeout)
	defer cancel()
	if u.deps.Health.Clear(clearCtx, u.raised) == nil {
		u.raised = ""
	}
}

func (u *Keep) adopt(ctx context.Context, revoked domain.DeviceRevokedError) error {
	if u.deps.Chains == nil {
		return nil
	}
	return u.deps.Chains.AdoptChain(ctx, revoked.Chain)
}

func (u *Keep) clearLeft(ctx context.Context) {
	if u.isClean {
		return
	}
	clearCtx, cancel := context.WithTimeout(ctx, hostCallTimeout)
	defer cancel()
	for _, what := range []concern{newNoIdentityConcern(), newServerUnreachableConcern(), newUpdateRequiredConcern(), newRevokedConcern(), newAccountDeletingConcern(), newSignInFailedConcern()} {
		if u.deps.Health.Clear(clearCtx, what.key) != nil {
			return
		}
	}
	u.isClean = true
}
