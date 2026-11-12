//go:build numen_debug

// Package debugserve serves a window's handler over HTTP without an OS window, for a test run of a
// debug build.
package debugserve

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
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

	"github.com/jiva-studio/numen/libs/atomicfile"
)

// Clock gets the current time.
type Clock interface {
	Now() time.Time
}

// Serve serves handler at addr until ctx is done or the process is interrupted.
// It serves TLS unless NUMEN_DEBUG_TLS is "false", with the PEM pair named by NUMEN_DEBUG_TLS_USE_CERT and NUMEN_DEBUG_TLS_USE_KEY (both or neither), else a new self-signed certificate.
// When set, it writes the certificate in PEM to the file NUMEN_DEBUG_TLS_CERT names and the key in PEM (PKCS #8) to the file NUMEN_DEBUG_TLS_KEY names.
// It writes the listening address to out once the listener is bound.
func Serve(
	ctx context.Context,
	clock Clock,
	addr string,
	handler http.Handler,
	out io.Writer,
) (result error) {
	server := &http.Server{Addr: addr, Handler: newHandler(handler)}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	// Shutdown closes the listener the server holds, so a second close reports it closed already.
	defer func() {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			result = errors.Join(result, fmt.Errorf("close listener: %w", err))
		}
	}()

	scheme := "http"
	if os.Getenv("NUMEN_DEBUG_TLS") != "false" {
		secure, err := createTLSListener(ln, clock)
		if err != nil {
			return err
		}
		ln = secure
		scheme = "https"
	}

	if _, err := fmt.Fprintf(out, "numen debug server listening on %s://%s\n", scheme, ln.Addr()); err != nil {
		return fmt.Errorf("write listening address: %w", err)
	}

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

func createTLSListener(ln net.Listener, clock Clock) (net.Listener, error) {
	cert, err := getCertificate(clock)
	if err != nil {
		return nil, err
	}
	if path := os.Getenv("NUMEN_DEBUG_TLS_CERT"); path != "" {
		if err := writePEMFile(path, "CERTIFICATE", cert.Certificate[0]); err != nil {
			return nil, fmt.Errorf("write debug tls cert: %w", err)
		}
	}
	if path := os.Getenv("NUMEN_DEBUG_TLS_KEY"); path != "" {
		key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("marshal debug tls key: %w", err)
		}
		if err := writePEMFile(path, "PRIVATE KEY", key); err != nil {
			return nil, fmt.Errorf("write debug tls key: %w", err)
		}
	}
	return tls.NewListener(ln, &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	}), nil
}

func getCertificate(clock Clock) (tls.Certificate, error) {
	certFile := os.Getenv("NUMEN_DEBUG_TLS_USE_CERT")
	keyFile := os.Getenv("NUMEN_DEBUG_TLS_USE_KEY")
	if certFile == "" && keyFile == "" {
		cert, err := generateSelfSignedCert(clock)
		if err != nil {
			return tls.Certificate{}, fmt.Errorf("generate debug tls cert: %w", err)
		}
		return cert, nil
	}
	if certFile == "" || keyFile == "" {
		return tls.Certificate{}, errors.New("NUMEN_DEBUG_TLS_USE_CERT and NUMEN_DEBUG_TLS_USE_KEY must be set together")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load debug tls cert: %w", err)
	}
	return cert, nil
}

func generateSelfSignedCert(clock Clock) (tls.Certificate, error) {
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
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		// A client trusts the certificate as its own root: the e2e scripts pass it as their only CA.
		IsCA: true,
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

func writePEMFile(path, blockType string, der []byte) error {
	return atomicfile.WriteFile(path, bytes.NewReader(pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})), 0o600)
}
