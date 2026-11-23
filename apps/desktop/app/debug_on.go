//go:build numen_debug

package app

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/debugserve"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// DebugOptions holds test settings for debug builds.
type DebugOptions struct {
	HTTPAddr   string
	IsMockML   bool
	FolderFile string
}

func RegisterDebugFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Debug.HTTPAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
	fs.StringVar(&opts.Debug.FolderFile, "debug-folder-file", "",
		"answer the folder dialog with the folder written in this file")
	fs.BoolVar(&opts.Debug.IsMockML, "debug-mock-ml", false,
		"use fast mock ML models, and an empty placeholder file for every dependency a plugin declares")
}

func (d *DebugOptions) apply(refreshDependencies func(), res *container.Resources) error {
	if d.isServingHTTP() {
		res.KeyStore = debugserve.NewKeyStore()
	}
	if !d.IsMockML {
		return nil
	}
	if err := os.Setenv("NUMEN_TEST_MOCK_ML", "true"); err != nil {
		return err
	}
	refreshDependencies()
	return nil
}

func (d *DebugOptions) ensureVault(cfg container.Config, res *container.Resources, vaultPath string) error {
	return debugserve.RegisterVault(cfg, res, vaultPath)
}

func (d *DebugOptions) isServingHTTP() bool {
	return d.HTTPAddr != ""
}

func (d *DebugOptions) serveHTTP(
	ctx context.Context,
	res *container.Resources,
	opened *editor.Installation,
	pages http.Handler,
) error {
	return debugserve.Serve(ctx, container.GetClock(res), d.HTTPAddr, opened.NewHandler(pages), os.Stdout)
}

func (d *DebugOptions) getDialog(window editor.PathDialog) editor.PathDialog {
	if !d.isServingHTTP() {
		return window
	}
	if d.FolderFile == "" {
		return nil
	}
	return debugserve.NewFolderFileDialog(d.FolderFile)
}
