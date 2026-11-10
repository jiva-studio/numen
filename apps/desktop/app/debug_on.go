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
	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// DebugOptions holds test settings for debug builds.
type DebugOptions struct {
	HTTPAddr string
	IsMockML bool
}

// RegisterDebugFlags registers debug flags on the flag set.
func RegisterDebugFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Debug.HTTPAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
	fs.BoolVar(&opts.Debug.IsMockML, "debug-mock-ml", false,
		"use fast mock ML models, and an empty placeholder file for every dependency a plugin declares")
}

// dependencySetter is a plugin host whose dependency catalog can be replaced.
type dependencySetter interface {
	SetDependencies(pluginport.DependencyCatalog)
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
	if host, ok := cfg.PluginHost.(dependencySetter); ok {
		host.SetDependencies(container.NewDependencies())
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
	opened *editor.Installation,
	pages http.Handler,
) error {
	opened.State.NotifyOpenTabs = func(domain.OpenTabs) {}
	opened.State.OpenVault = func(ctx context.Context, v domain.Vault, open *editor.PluginOpen) error {
		if err := opened.Show(ctx, v, open); err != nil {
			return err
		}
		cfg.SetPluginVaultPath(v.Path)
		return nil
	}
	return debugserve.Serve(ctx, cfg.Clock(), d.HTTPAddr, opened.NewHandler(pages), os.Stdout)
}
