//go:build debug

package main

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

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agents"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// debugOptions holds test settings for debug builds.
type debugOptions struct {
	httpAddr string
	isMockML bool
}

func registerDebugFlags(fs *flag.FlagSet) *debugOptions {
	d := &debugOptions{}
	fs.StringVar(&d.httpAddr, "debug-http-addr", "", "serve the window over HTTP without opening an OS window")
	fs.BoolVar(&d.isMockML, "debug-mock-ml", false, "use fast mock ML models for automated testing")
	return d
}

func (d *debugOptions) apply(cfg *container.Config) {
	if d.isMockML || os.Getenv("NUMEN_MOCK_ML") == "true" {
		cfg.IsMockML = true
	}
}

func (d *debugOptions) ensureVault(cfg container.Config, vaultPath string) error {
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

func (d *debugOptions) isServingHTTP() bool {
	return d.httpAddr != ""
}

func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Numen Debug"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}, nil
}

func (d *debugOptions) serveHTTP(
	ctx context.Context,
	cfg container.Config,
	opened *editor.Installation,
	pages http.Handler,
	reachable *agents.Endpoint,
) error {
	reachable.Start()

	listener, err := net.Listen("tcp", d.httpAddr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", d.httpAddr, err)
	}
	defer listener.Close()

	cert, err := generateSelfSignedCert()
	if err != nil {
		return fmt.Errorf("generate cert: %w", err)
	}
	tlsListener := tls.NewListener(listener, &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	})

	handler := opened.Services.NewHandler(pages)
	server := &http.Server{
		Handler: handler,
	}

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stdout, "numen debug server listening on https://%s\n", listener.Addr().String())
		if err := server.Serve(tlsListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-sigCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
