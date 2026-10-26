//go:build numen_debug

package app

import (
	"context"
	"flag"
	"net/http"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/debugserve"
	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// DebugOptions holds test settings for debug builds.
type DebugOptions struct {
	HttpAddr  string
	IsMockML  bool
	UpdateURL string
}

// RegisterDebugFlags registers debug flags on the flag set.
func RegisterDebugFlags(fs *flag.FlagSet, opts *Options) {
	fs.StringVar(&opts.Debug.HttpAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
	fs.BoolVar(&opts.Debug.IsMockML, "debug-mock-ml", false,
		"use fast mock ML models, and an empty placeholder file for every dependency a plugin declares")
	fs.StringVar(&opts.Debug.UpdateURL, "debug-update-url", "", "where the release manifest is asked for")
}

// dependencySetter is a plugin host whose dependency catalog can be replaced.
type dependencySetter interface {
	SetDependencies(pluginport.DependencyCatalog)
}

// apply has the plugins this run starts answer with mock models where the flag
// asks for them. The plugin host was made before the flag was read, so it is
// handed the dependencies a run on mock models provisions: placeholders, and
// nothing fetched.
func (d *DebugOptions) apply(cfg container.Config) error {
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
	return d.HttpAddr != ""
}

func (d *DebugOptions) serveHTTP(
	ctx context.Context,
	cfg container.Config,
	opened *editor.Installation,
	pages http.Handler,
) error {
	if d.UpdateURL != "" {
		opened.Services.UpdateEndpoint = d.UpdateURL
	}
	opened.State.NotifyOpenTabs = func(domain.OpenTabs) {}
	opened.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		cfg.SetPluginVaultPath(v.Path)
		return opened.Show(ctx, v)
	}
	return debugserve.Serve(ctx, d.HttpAddr, opened.Services.NewHandler(pages), os.Stdout)
}
