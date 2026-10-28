//go:build numen_debug

// Package debugserve serves a window's handler over HTTP without an OS window, for a test run of a
// debug build.
package debugserve

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jiva-studio/numen/apps/desktop/internal/adapter/clock"
)

// Serve answers with handler at addr until ctx is done or the process is interrupted. It serves TLS
// on a certificate of its own unless NUMEN_DEBUG_TLS is "false", and writes the address it listens on
// to out once it does.
func Serve(ctx context.Context, addr string, handler http.Handler, out io.Writer) error {
	server := &http.Server{Addr: addr, Handler: newHandler(handler)}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	defer ln.Close()

	scheme := "http"
	if os.Getenv("NUMEN_DEBUG_TLS") != "false" {
		cert, err := generateSelfSignedCert()
		if err != nil {
			return fmt.Errorf("generate debug tls cert: %w", err)
		}
		ln = tls.NewListener(ln, &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"h2", "http/1.1"},
		})
		scheme = "https"
	}

	fmt.Fprintf(out, "numen debug server listening on %s://%s\n", scheme, ln.Addr())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

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
		NotBefore:             clock.Now().Add(-1 * time.Hour),
		NotAfter:              clock.Now().Add(24 * time.Hour),
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
