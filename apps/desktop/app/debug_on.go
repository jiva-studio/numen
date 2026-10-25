//go:build debug

package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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

// apply sets the configuration the debug flags name. The plugin processes are launched with the
// host's environment, so the mock models reach them through NUMEN_MOCK_ML.
func (d *DebugOptions) apply(cfg *container.Config) error {
	if d.IsMockML || os.Getenv("NUMEN_MOCK_ML") == "true" {
		cfg.IsMockML = true
		return os.Setenv("NUMEN_MOCK_ML", "true")
	}
	return nil
}

func (d *DebugOptions) ensureVault(cfg container.Config, vaultPath string) error {
	if vaultPath == "" {
		return nil
	}
	abs, err := filepath.Abs(vaultPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil
	}
	reg, err := cfg.Registry()
	if err != nil {
		return err
	}
	_, found, err := reg.FindVault(abs)
	if err != nil {
		return err
	}
	if !found {
		v := domain.Vault{
			ID:   domain.VaultID("01E2ETESTVAULT000000000000"),
			Name: filepath.Base(abs),
			Path: abs,
		}
		if err := reg.Save(v); err != nil {
			return err
		}
		if err := reg.RecordOpened(v.ID); err != nil {
			return err
		}
	}
	return nil
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
	server := &http.Server{
		Addr:    d.HttpAddr,
		Handler: opened.Services.NewHandler(pages),
	}

	ln, err := net.Listen("tcp", d.HttpAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", d.HttpAddr, err)
	}
	defer ln.Close()

	var tlsConf *tls.Config
	if os.Getenv("NUMEN_DEBUG_TLS") != "false" {
		cert, err := generateSelfSignedCert()
		if err != nil {
			return fmt.Errorf("generate debug tls cert: %w", err)
		}
		tlsConf = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		}
		ln = tls.NewListener(ln, tlsConf)
	}

	naming := func(open domain.OpenTabs) {}
	opened.State.NotifyOpenTabs = naming
	opened.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		cfg.SetPluginVaultPath(v.Path)
		err := opened.Show(ctx, v)
		naming(opened.State.GetOpenTabs())
		return err
	}

	scheme := "http"
	if tlsConf != nil {
		scheme = "https"
	}
	fmt.Fprintf(os.Stdout, "numen debug server listening on %s://%s\n", scheme, ln.Addr())
	_ = os.Stdout.Sync()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case <-sigCh:
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Numen Debug"},
			CommonName:   "127.0.0.1",
		},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}, nil
}
