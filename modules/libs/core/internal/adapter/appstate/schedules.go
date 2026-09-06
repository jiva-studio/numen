package appstate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
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
	at, err := s.at(vaultID)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(at)
}

func (s *Schedules) Write(_ context.Context, vaultID domain.VaultID, content []byte) error {
	at, err := s.at(vaultID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return err
	}

	// The file is written whole beside itself and moved into place, so a second
	// window writing the same vault leaves one of the two and never half of
	// both. A write that fails anywhere takes its temporary file away with it.
	tmp, err := os.CreateTemp(filepath.Dir(at), filepath.Base(at)+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), at); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// at is the file one vault's schedules stand in.
//
// The identity is written into the name, and an identity carrying a separator
// or a dot is refused.
func (s *Schedules) at(vaultID domain.VaultID) (string, error) {
	name := string(vaultID)
	if name == "" || strings.ContainsAny(name, `/\.`) {
		return "", fmt.Errorf("%q is not the identity of a vault", name)
	}
	return filepath.Join(s.dir, name+".json"), nil
}
