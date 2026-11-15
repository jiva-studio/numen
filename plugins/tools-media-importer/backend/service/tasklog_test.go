package service

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/queuetest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

func waitForFetch(t *testing.T, tasks *queuetest.Log, path string) {
	t.Helper()
	tasks.WaitForEnd(t, "text:Mills:"+path)
	tasks.WaitForEnd(t, link.GetCopyTaskID("Mills", path))
}
