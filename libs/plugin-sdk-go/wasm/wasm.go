// Package wasm holds what a plugin that runs WebAssembly shares: the compilation cache of the
// runtime. The host grants the plugin the directory the cache lives in.
package wasm

import (
	"path/filepath"

	"github.com/tetratelabs/wazero"
)

// cacheFolder is the folder of the compilation cache inside the directory the host granted.
const cacheFolder = "wazero"

// NewCompilationCache returns the cache a WebAssembly module is kept compiled in. dir is the cache
// directory the host granted the plugin, which only the host and the plugin can write to: the cache
// holds machine code that the runtime loads without checking it. A directory that cannot be used
// leaves the compiled module in memory for the life of the process.
func NewCompilationCache(dir string) wazero.CompilationCache {
	if dir != "" {
		if diskCache, err := wazero.NewCompilationCacheWithDir(filepath.Join(dir, cacheFolder)); err == nil {
			return diskCache
		}
	}
	return wazero.NewCompilationCache()
}
