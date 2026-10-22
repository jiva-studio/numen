package engine

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

// IsFetched says whether one recogniser model file is on this machine, by the
// same rule model() finds it.
func IsFetched(cfg Config, held RecogniserModel) bool {
	if held.Path != "" {
		return isPresent(held.Path)
	}
	if held.Name == "" {
		return false
	}
	if _, ok := models.FindLocal(cfg.source(), pinnedSums, held.Name); ok {
		return true
	}
	if !models.IsAddress(held.Name) {
		return false
	}
	dir, err := cfg.folder().GetDir()
	if err != nil {
		return false
	}
	sum := pinnedSums.GetSum(held.Name)
	pos := filepath.Join(dir, models.GetCacheName(held.Name, sum))
	if !isPresent(pos) {
		return false
	}
	return sum == "" || models.VerifyFile(pos, sum) == nil
}

func isPresent(pos string) bool {
	_, err := os.Stat(pos)
	return err == nil
}
