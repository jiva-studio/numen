package engine

import (
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
)

// IsFetched says whether one recogniser model file is on this machine, by the
// same rule model() finds it.
func IsFetched(cfg Config, model RecogniserModel) bool {
	if model.Path != "" {
		return isPresent(model.Path)
	}
	if model.Name == "" {
		return false
	}
	if _, ok := models.FindLocal(cfg.source(), pinnedSums, model.Name); ok {
		return true
	}
	if !models.IsAddress(model.Name) {
		return false
	}
	dir, err := cfg.folder().GetDir()
	if err != nil {
		return false
	}
	sum := pinnedSums.GetSum(model.Name)
	pos := filepath.Join(dir, models.GetCacheName(model.Name, sum))
	if !isPresent(pos) {
		return false
	}
	return sum == "" || models.VerifyFile(pos, sum) == nil
}

func isPresent(pos string) bool {
	_, err := os.Stat(pos)
	return err == nil
}
