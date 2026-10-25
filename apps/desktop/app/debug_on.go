//go:build debug

package app

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/debugserve"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// DebugOptions holds test settings for debug builds.
type DebugOptions struct {
	HttpAddr string
	IsMockML bool
}

// RegisterDebugFlags registers debug flags on the flag set.
func RegisterDebugFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Debug.HttpAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
	fs.BoolVar(&opts.Debug.IsMockML, "debug-mock-ml", false, "use fast mock ML models for automated testing")
}

// apply has the plugins this run starts answer with mock models where the flag asks for them.
func (d *DebugOptions) apply() error {
	if !d.IsMockML {
		return nil
	}
	return os.Setenv("NUMEN_MOCK_ML", "true")
}

func (d *DebugOptions) ensureVault(cfg container.Config, vaultPath string) error {
	return debugserve.RegisterVault(cfg, vaultPath)
}

func (d *DebugOptions) isServingHTTP() bool {
	return d.HttpAddr != ""
}

func (d *DebugOptions) serveHTTP(
	ctx context.Context,
	cfg container.Config,
	opened *editor.Installation,
	pages http.Handler,
) error {
	opened.State.NotifyOpenTabs = func(domain.OpenTabs) {}
	opened.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		cfg.SetPluginVaultPath(v.Path)
		return opened.Show(ctx, v)
	}
	return debugserve.Serve(ctx, d.HttpAddr, opened.Services.NewHandler(pages), os.Stdout)
}
