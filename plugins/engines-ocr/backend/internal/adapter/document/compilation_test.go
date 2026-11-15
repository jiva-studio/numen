package document_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
)

func TestCachedReaderCompilesPDFiumIntoThePluginCacheFolder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(cache.EnvCacheDir, dir)

	document.NewCachedReader()

	entries, err := os.ReadDir(filepath.Join(dir, "wazero"))
	if err != nil || len(entries) == 0 {
		t.Errorf("the plugin cache folder %s holds no compilation cache: %v, %v", dir, entries, err)
	}
}
