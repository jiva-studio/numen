//go:build numen_debug

package debugserve

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func newTestClock() fixedClock {
	return fixedClock{now: time.Date(2026, 3, 14, 9, 30, 0, 0, time.UTC)}
}

func writeKeyPair(t *testing.T, dir string) (certFile, keyFile string, der []byte) {
	t.Helper()
	cert, err := generateSelfSignedCert(newTestClock())
	if err != nil {
		t.Fatalf("generate certificate: %v", err)
	}
	key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	certFile = filepath.Join(dir, "given-certificate.pem")
	keyFile = filepath.Join(dir, "given-key.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key})
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatalf("write certificate: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return certFile, keyFile, cert.Certificate[0]
}

func startServe(t *testing.T) string {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	reader, writer := io.Pipe()
	served := make(chan error, 1)
	go func() {
		err := Serve(ctx, newTestClock(), "127.0.0.1:0", http.NotFoundHandler(), writer)
		served <- errors.Join(err, writer.CloseWithError(err))
	}()
	lines := bufio.NewReader(reader)
	line, err := lines.ReadString('\n')
	if err != nil {
		cancel()
		t.Fatalf("read the listening address: %v", err)
	}
	drained := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, lines)
		drained <- err
	}()
	t.Cleanup(func() {
		cancel()
		if err := <-served; err != nil {
			t.Errorf("serve: %v", err)
		}
		if err := <-drained; err != nil {
			t.Errorf("read the server output: %v", err)
		}
	})
	return strings.TrimPrefix(strings.TrimSpace(line), "numen debug server listening on https://")
}

func readServedCertificate(t *testing.T, addr string, trusted []byte) []byte {
	t.Helper()
	parsed, err := x509.ParseCertificate(trusted)
	if err != nil {
		t.Fatalf("parse trusted certificate: %v", err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(parsed)
	dialer := &tls.Dialer{Config: &tls.Config{RootCAs: roots, ServerName: "127.0.0.1", Time: newTestClock().Now}}
	conn, err := dialer.DialContext(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatalf("connect trusting only the expected certificate: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close connection: %v", err)
		}
	}()
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		t.Fatalf("connection is %T, not TLS", conn)
	}
	return tlsConn.ConnectionState().PeerCertificates[0].Raw
}

func TestServeServesGivenCertificateAndKey(t *testing.T) {
	certFile, keyFile, der := writeKeyPair(t, t.TempDir())
	t.Setenv("NUMEN_DEBUG_TLS_USE_CERT", certFile)
	t.Setenv("NUMEN_DEBUG_TLS_USE_KEY", keyFile)

	addr := startServe(t)

	if served := readServedCertificate(t, addr, der); !bytes.Equal(served, der) {
		t.Fatal("the server presents another certificate than the one it was given")
	}
}

func TestServeWritesGivenCertificateToCertificateFile(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, der := writeKeyPair(t, dir)
	outputCertFile := filepath.Join(dir, "own-certificate.pem")
	t.Setenv("NUMEN_DEBUG_TLS_USE_CERT", certFile)
	t.Setenv("NUMEN_DEBUG_TLS_USE_KEY", keyFile)
	t.Setenv("NUMEN_DEBUG_TLS_CERT", outputCertFile)

	startServe(t)

	data, err := os.ReadFile(outputCertFile)
	if err != nil {
		t.Fatalf("read the written certificate: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || !bytes.Equal(block.Bytes, der) {
		t.Fatal("the written certificate is not the one the server was given")
	}
}

func TestServeWritesKeyNextToCertificate(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "certificate.pem")
	keyFile := filepath.Join(dir, "key.pem")
	t.Setenv("NUMEN_DEBUG_TLS_CERT", certFile)
	t.Setenv("NUMEN_DEBUG_TLS_KEY", keyFile)

	addr := startServe(t)

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("load the written certificate and key as a pair: %v", err)
	}
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("key file mode = %o, want 600", mode)
	}
	if served := readServedCertificate(t, addr, pair.Certificate[0]); !bytes.Equal(served, pair.Certificate[0]) {
		t.Fatal("the server presents another certificate than the one it wrote")
	}
}

func TestServeRefusesACertificateWithoutItsKey(t *testing.T) {
	certFile, _, _ := writeKeyPair(t, t.TempDir())
	t.Setenv("NUMEN_DEBUG_TLS_USE_CERT", certFile)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	err := Serve(ctx, newTestClock(), "127.0.0.1:0", http.NotFoundHandler(), io.Discard)

	if err == nil {
		t.Fatal("Serve with a certificate and no key returned no error")
	}
}

func TestServeWritesKeyReadableByItsOwnerOverALeftoverPartFile(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "certificate.pem")
	keyFile := filepath.Join(dir, "key.pem")
	if err := os.WriteFile(keyFile+".part", []byte("left from an earlier run\n"), 0o644); err != nil {
		t.Fatalf("write the leftover part file: %v", err)
	}
	t.Setenv("NUMEN_DEBUG_TLS_CERT", certFile)
	t.Setenv("NUMEN_DEBUG_TLS_KEY", keyFile)

	startServe(t)

	info, err := os.Lstat(keyFile)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("key file mode = %o, want 600", mode)
	}
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		t.Fatalf("load the written certificate and key as a pair: %v", err)
	}
}

func TestServeDoesNotWriteTheKeyThroughASymlinkedPartFile(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "certificate.pem")
	keyFile := filepath.Join(dir, "key.pem")
	target := filepath.Join(dir, "unrelated.txt")
	const unrelated = "a file the server does not own\n"
	if err := os.WriteFile(target, []byte(unrelated), 0o644); err != nil {
		t.Fatalf("write the unrelated file: %v", err)
	}
	if err := os.Symlink(target, keyFile+".part"); err != nil {
		t.Fatalf("link the part file: %v", err)
	}
	t.Setenv("NUMEN_DEBUG_TLS_CERT", certFile)
	t.Setenv("NUMEN_DEBUG_TLS_KEY", keyFile)

	startServe(t)

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read the unrelated file: %v", err)
	}
	if string(data) != unrelated {
		t.Fatal("the server wrote the key through the symlink into the unrelated file")
	}
	info, err := os.Lstat(keyFile)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode = %v, want a regular file with mode 600", info.Mode())
	}
	if _, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
		t.Fatalf("load the written certificate and key as a pair: %v", err)
	}
}
