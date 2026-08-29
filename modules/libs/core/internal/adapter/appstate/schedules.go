package appstate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Schedules keeps what a review worked out, one file to a vault, in the
// platform's cache location.
//
// It is a cache in the place the platform keeps caches, so a person clearing it
// costs themselves a replay of the answers and nothing else.
type Schedules struct{ dir string }

// OpenSchedules uses the platform's cache location.
func OpenSchedules() (*Schedules, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return SchedulesAt(filepath.Join(dir, "numen", "review")), nil
}

// SchedulesAt is OpenSchedules with an explicit folder, so that a test does not
// touch the machine's own.
func SchedulesAt(dir string) *Schedules { return &Schedules{dir: dir} }

func (s *Schedules) Read(_ context.Context, vaultID string) ([]byte, error) {
	at, err := s.at(vaultID)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(at)
}

func (s *Schedules) Write(_ context.Context, vaultID string, content []byte) error {
	at, err := s.at(vaultID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return err
	}
	return os.WriteFile(at, content, 0o644)
}

// at is the file one vault's schedules stand in.
//
// The identity is a ULID and is written into the name, so it is checked for
// being one: a name arriving from anywhere else could otherwise reach a file
// this folder does not hold.
func (s *Schedules) at(vaultID string) (string, error) {
	if vaultID == "" || strings.ContainsAny(vaultID, `/\.`) {
		return "", fmt.Errorf("%q is not the identity of a vault", vaultID)
	}
	return filepath.Join(s.dir, vaultID+".json"), nil
}
