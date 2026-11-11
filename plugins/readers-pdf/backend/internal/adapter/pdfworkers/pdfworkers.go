// Package pdfworkers starts the workers the plugin reads PDF files on, with the compiled PDFium
// kept in the plugin's cache folder.
package pdfworkers

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/tetratelabs/wazero"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/wasm"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
)

// ReportError writes a background failure of the plugin to standard error.
func ReportError(what string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "readers-pdf: %s: %v\n", what, err)
	}
}

func newCompilationCache() wazero.CompilationCache {
	dir, err := cache.GetPluginDir()
	// gomutants:disable-next-line BRANCH_IF reason="missing plugin cache directory falls back to in-memory compilation cache"
	if err != nil {
		return wasm.NewCompilationCache("")
	}
	return wasm.NewCompilationCache(dir)
}

// New starts one worker for each processor and warms them up. afterFunc sets the timers that
// release idle workers.
func New(afterFunc func(time.Duration, func()) pdf.Timer) *pdf.Workers {
	workers := pdf.NewWorkers()
	// gomutants:disable-next-line STATEMENT_REMOVE reason="production workers share the plugin compilation cache"
	workers.SetCompilationCache(newCompilationCache())
	workers.SetAfterFunc(afterFunc)
	workers.SetReport(ReportError)
	workers.SetWorkers(runtime.NumCPU())
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="idle worker count is bounded between 2 and 4"
	workers.SetIdleWorkers(max(2, min(runtime.NumCPU(), 4)))
	go workers.Warmup()
	return workers
}
