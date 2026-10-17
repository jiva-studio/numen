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
	if _, ok := findLocal(cfg, held.Name); ok {
		return true
	}
	if !models.IsAddress(held.Name) {
		return false
	}
	dir, err := cfg.source().GetDir()
	if err != nil {
		return false
	}
	sum := getPinnedSum(held.Name)
	at := filepath.Join(dir, models.GetCacheName(held.Name, sum))
	if !isPresent(at) {
		return false
	}
	return sum == "" || models.VerifyFile(at, sum) == nil
}

func isPresent(at string) bool {
	_, err := os.Stat(at)
	return err == nil
}
