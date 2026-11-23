// Package deletions pauses the sync before it sends or applies the removal of many files, and carries out the
// answer of the person.
package deletions

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

const HealthKey = "deletions-paused"

var errNotImplemented = errors.New("not implemented")

type Direction string

const (
	DirectionSend  Direction = "send"
	DirectionApply Direction = "apply"
)

type Verdict int

const (
	Go Verdict = iota + 1
	Pause
)

type Choice string

const (
	ChoiceRemove Choice = "remove"
	ChoiceKeep   Choice = "keep"
)

// Removal is a file a pass is about to remove on the server or in the vault.
type Removal struct {
	FileID   []byte `json:"fileId"`
	Path     string `json:"path"`
	BaseHash string `json:"baseHash"`
}

// State is the pending answer of one vault, kept in the store of the plugin.
type State struct {
	Direction Direction `json:"direction"`
	Files     []Removal `json:"files"`
	Tracked   int       `json:"tracked"`
	CreatedAt time.Time `json:"createdAt"`
}

// Pending keeps the state of the vaults. ReadPending returns nil where nothing is pending.
type Pending interface {
	ReadPending(ctx context.Context, id domain.VaultID) (*State, error)
	WritePending(ctx context.Context, id domain.VaultID, state State) error
	DeletePending(ctx context.Context, id domain.VaultID) error
}

// Health is where the plugin says what does not work.
type Health interface {
	Report(ctx context.Context, item health.Item) error
	Clear(ctx context.Context, key string) error
}

// Clock reads the time.
type Clock interface {
	Now() time.Time
}

type Deps struct {
	Pending Pending
	Health  Health
	Clock   Clock
}

type Input struct {
	Vault      domain.VaultID
	Direction  Direction
	Candidates []Removal
	Tracked    int
}

// GuardDeletions decides whether a pass may remove files.
type GuardDeletions struct{ deps Deps }

func NewGuardDeletions(deps Deps) *GuardDeletions { return &GuardDeletions{deps: deps} }

// Execute returns Pause, and keeps the state until it is answered, when the candidates are at least
// note.MassDeleteMinFiles and more than note.MassDeleteRatio percent of the tracked files.
func (u *GuardDeletions) Execute(ctx context.Context, in Input) (Verdict, error) {
	return 0, errNotImplemented
}

// ResolveDeletions carries out the answer of the person for one vault.
type ResolveDeletions struct{ deps Deps }

func NewResolveDeletions(deps Deps) *ResolveDeletions { return &ResolveDeletions{deps: deps} }

// Execute removes the pending state and the health item and returns the state the answer was for.
func (u *ResolveDeletions) Execute(ctx context.Context, id domain.VaultID, choice Choice) (State, error) {
	return State{}, errNotImplemented
}
