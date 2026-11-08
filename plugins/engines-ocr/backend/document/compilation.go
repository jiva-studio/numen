package document

import (
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
)

// SetCompilationCache makes the worker that starts afterwards compile PDFium into cache, so readers
// that share a cache compile it once. It has no effect on a reader whose worker has started.
func (r *Reader) SetCompilationCache(cache wazero.CompilationCache) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.compilationCache = cache
}

func (r *Reader) getRuntimeConfig() wazero.RuntimeConfig {
	if r.compilationCache == nil {
		return nil
	}
	return wazero.NewRuntimeConfig().
		WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesExceptionHandling).
		WithCompilationCache(r.compilationCache)
}
