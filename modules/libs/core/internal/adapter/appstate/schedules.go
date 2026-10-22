package appstate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Schedules keeps what a replay of the answers worked out, one file to a vault,
// in the platform's cache location.
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
	return SchedulesAt(filepath.Join(dir, "numen", "flashcards")), nil
}

// OpenCounting is where what each day of review came to is kept, beside the
// schedules and under a folder of its own.
func OpenCounting() (*Schedules, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return SchedulesAt(filepath.Join(dir, "numen", "flashcards", "days")), nil
}

// SchedulesAt is OpenSchedules with an explicit folder, so that a test does not
// touch the machine's own.
func SchedulesAt(dir string) *Schedules { return &Schedules{dir: dir} }

func (s *Schedules) Read(_ context.Context, vaultID domain.VaultID) ([]byte, error) {
	pos, err := s.getPath(vaultID)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(pos)
}

// Write replaces the vault's file whole, so a second window writing the same
// vault leaves one of the two and never half of both.
func (s *Schedules) Write(_ context.Context, vaultID domain.VaultID, content []byte) error {
	pos, err := s.getPath(vaultID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(pos), privateDirMode); err != nil {
		return err
	}
	return replace(pos, content)
}

// getPath is the file one vault's schedules stand in.
//
// The identity is written into the name, and an identity carrying a separator
func (s *Schedules) getPath(vaultID domain.VaultID) (string, error) {
	name := string(vaultID)
	if name == "" || strings.ContainsAny(name, `/\.`) {
		return "", fmt.Errorf("%q is not the identity of a vault", name)
	}
	return filepath.Join(s.dir, name+".json"), nil
}
