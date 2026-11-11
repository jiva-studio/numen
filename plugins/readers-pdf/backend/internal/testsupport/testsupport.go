// Package testsupport holds the helpers the tests of the backend packages share: PDF workers over a
// compilation cache made once for a test binary, the fixture files and the writer of text layers over
// a test vault.
package testsupport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tetratelabs/wazero"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/pdf"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

const PluginID = "md.numen.readers.pdf"

// compilationDirVariable names the folder Run makes for the compiled PDFium of the test binary.
const compilationDirVariable = "NUMEN_PDF_TEST_COMPILATION_DIR"

// Run runs the tests of m with a folder for the compiled PDFium and returns the exit code. A package
// calls it from TestMain.
func Run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "numen-pdf-compilation-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "make the compilation folder: %v\n", err)
		return 1
	}
	if err := os.Setenv(compilationDirVariable, dir); err != nil {
		fmt.Fprintf(os.Stderr, "set %s: %v\n", compilationDirVariable, err)
		return 1
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintf(os.Stderr, "remove the compilation folder: %v\n", err)
		code = 1
	}
	return code
}

// NewWorkers returns PDF workers that keep the compiled PDFium in the folder Run made, closed when
// the test ends.
func NewWorkers(tb testing.TB) *pdf.Workers {
	tb.Helper()
	compilation, err := wazero.NewCompilationCacheWithDir(os.Getenv(compilationDirVariable))
	if err != nil {
		tb.Fatal(err)
	}
	workers := pdf.NewWorkers(pdf.Options{
		Report: func(what string, err error) { tb.Errorf("%s: %v", what, err) }, CompilationCache: compilation,
		AfterFunc: nil, Workers: 0, IdleWorkers: 0,
	})
	tb.Cleanup(func() {
		if err := errors.Join(workers.Close(), compilation.Close(context.WithoutCancel(tb.Context()))); err != nil {
			tb.Error(err)
		}
	})
	return workers
}

// ReadFixture returns the bytes of a PDF file of the testdata folder of the pdf package.
func ReadFixture(tb testing.TB, name string) []byte {
	tb.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("find the folder of the test support")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(here), "..", "..", "pdf", "testdata", name))
	if err != nil {
		tb.Fatal(err)
	}
	return raw
}

// FilesOf returns the vaults of vault as the files the writer reads.
func FilesOf(vault *hosttest.Vault) func(domain.VaultID) extract.Files {
	return func(id domain.VaultID) extract.Files { return vault.With(id) }
}

// NewWriter returns a writer that reads files through vaults, writes into the store of home and
// reports to texts, closed when the test ends.
func NewWriter(t *testing.T, vaults func(domain.VaultID) extract.Files, home *hosttest.Vault, texts *hosttest.Texts, queue host.Queue) *extract.Writer {
	t.Helper()
	writer := extract.NewWriter(
		vaults,
		func(id domain.VaultID) host.TextVersions { return texts.With(id) },
		func(id domain.VaultID) host.StoreWriter { return home.Store().With(id) },
		queue, NewWorkers(t), func(what string, err error) { t.Logf("%s: %v", what, err) })
	t.Cleanup(writer.Close)
	return writer
}

// NewVault returns a vault holding the PDF fixture under name, the texts of the vault, and the
// writer of its text layers.
func NewVault(t *testing.T, name, fixture string) (*hosttest.Texts, *extract.Writer, []byte) {
	t.Helper()
	return NewVaultOf(t, name, ReadFixture(t, fixture))
}

// NewVaultOf returns a vault holding raw under name, the texts of the vault, and the writer of its
// text layers.
func NewVaultOf(t *testing.T, name string, raw []byte) (*hosttest.Texts, *extract.Writer, []byte) {
	t.Helper()
	files := hosttest.OnDisk(t, "vault", map[string]string{name: string(raw)})
	texts := hosttest.NewTexts("vault", PluginID)
	texts.File(name, raw)
	texts.SetStore(PluginID, files.Store())
	return texts, NewWriter(t, FilesOf(files), files, texts, hosttest.NewQueue()), raw
}

// CreateText writes the text layer of the file req names and waits for it.
func CreateText(t *testing.T, writer *extract.Writer, req extract.Request) {
	t.Helper()
	if err := writer.CreateText(t.Context(), req); err != nil {
		t.Fatalf("CreateText: %v", err)
	}
	writer.Wait()
}

func ReadLive(t *testing.T, texts *hosttest.Texts, path string) domain.VersionText {
	t.Helper()
	listing, err := texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range listing.Versions {
		if v.GetID() == listing.Live {
			text, err := host.ReadWholeText(t.Context(), texts, v)
			if err != nil {
				t.Fatal(err)
			}
			return text
		}
	}
	t.Fatalf("%s has no live version among %v", path, listing.Versions)
	return domain.VersionText{}
}
