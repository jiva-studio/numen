package document

import (
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/wasm"
)

// NewPluginReader returns a reader that keeps the compiled PDFium in the cache folder the host granted
// the plugin, so a plugin process that starts again loads PDFium instead of compiling it. Without a
// granted folder PDFium stays compiled in memory for the life of the process.
func NewPluginReader() *Reader {
	reader := NewReader()
	dir, err := cache.GetPluginDir()
	if err != nil {
		dir = ""
	}
	reader.SetCompilationCache(wasm.NewCompilationCache(dir))
	return reader
}

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
