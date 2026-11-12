//go:build numen_debug

package app

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/debugserve"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// DebugOptions holds test settings for debug builds.
type DebugOptions struct {
	HTTPAddr string
	IsMockML bool
	// FolderFile names a file holding the folder the folder dialog answers with.
	FolderFile string
}

func RegisterDebugFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Debug.HTTPAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
	fs.StringVar(&opts.Debug.FolderFile, "debug-folder-file", "",
		"answer the folder dialog with the folder written in this file")
	fs.BoolVar(&opts.Debug.IsMockML, "debug-mock-ml", false,
		"use fast mock ML models, and an empty placeholder file for every dependency a plugin declares")
}

func (d *DebugOptions) apply(cfg *container.Config) error {
	if d.isServingHTTP() {
		cfg.KeyStore = debugserve.NewKeyStore()
	}
	if !d.IsMockML {
		return nil
	}
	if err := os.Setenv("NUMEN_MOCK_ML", "true"); err != nil {
		return err
	}
	if cfg.PluginHost != nil {
		cfg.PluginHost.RefreshDependencies()
	}
	return nil
}

func (d *DebugOptions) ensureVault(cfg container.Config, vaultPath string) error {
	return debugserve.RegisterVault(cfg, vaultPath)
}

func (d *DebugOptions) isServingHTTP() bool {
	return d.HTTPAddr != ""
}

func (d *DebugOptions) serveHTTP(
	ctx context.Context,
	cfg container.Config,
	opened *container.OpenedWindow,
	pages http.Handler,
) error {
	if d.FolderFile != "" {
		opened.SetFolderDialog(debugserve.NewFolderFileDialog(d.FolderFile))
	}
	return debugserve.Serve(ctx, cfg.Clock(), d.HTTPAddr, opened.NewHandler(pages), os.Stdout)
}
