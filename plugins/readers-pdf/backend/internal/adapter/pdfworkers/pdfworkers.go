// Package pdfworkers starts the workers the plugin reads PDF files on, with the compiled PDFium
// kept in the plugin's cache folder.
package pdfworkers

import (
	"fmt"
	"os"
	"time"

	"github.com/tetratelabs/wazero"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
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

// New starts as many workers as the fallback performance limits allow threads and warms them up,
// keeping every returned worker alive. afterFunc sets the timers that release idle workers.
func New(afterFunc func(time.Duration, func()) pdf.Timer) *pdf.Workers {
	threads := performance.Read(nil).Threads
	workers := pdf.NewWorkers(pdf.Options{
		Report: ReportError,
		// gomutants:disable-next-line STATEMENT_REMOVE reason="production workers share the plugin compilation cache"
		CompilationCache: newCompilationCache(),
		StartTimer:        afterFunc,
		Workers:          threads,
		IdleWorkers:      threads,
	})
	go workers.WarmWorkers()
	return workers
}

// Configure sizes the workers, and the returned workers kept alive, to the threads the performance
// limits in the values of a Configure call allow.
func Configure(workers *pdf.Workers, values map[string]string) {
	threads := performance.Read(values).Threads
	workers.SetWorkers(threads, threads)
}
