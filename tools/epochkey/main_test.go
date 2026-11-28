package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testNow = func() time.Time { return time.Unix(1_800_000_000, 0) }

func runTool(tb testing.TB, now func() time.Time, args ...string) (string, error) {
	tb.Helper()
	var out bytes.Buffer
	err := run(args, &out, now)
	return strings.TrimSpace(out.String()), err
}

func newEpochKeyHex(tb testing.TB) string {
	tb.Helper()
	public, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		tb.Fatal(err)
	}
	return hex.EncodeToString(public)
}

func TestKeygen_WritesThePrivateKeyWithModeZeroSixHundredAndPrintsThePublicKey(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "root.key")

	public, err := runTool(t, testNow, "keygen", "-out", path)

	if err != nil || len(public) != 64 {
		t.Fatalf("keygen = %q %v, want 64 hex digits", public, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("key file = %v %v, want mode 0600", info, err)
	}
	if _, err := runTool(t, testNow, "keygen", "-out", path); err == nil {
		t.Error("a second keygen over the same file succeeded, want it refused")
	}
}

func TestSignCert_ProducesACertificateThatVerifyCertAcceptsUntilItExpires(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "root.key")
	public, err := runTool(t, testNow, "keygen", "-out", rootPath)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := runTool(t, testNow, "sign-cert", "-root", rootPath, "-epoch-key", newEpochKeyHex(t), "-serial", "1", "-days", "30")
	if err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(dir, "cert.json")
	if err := os.WriteFile(certPath, []byte(cert), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := runTool(t, testNow, "verify-cert", "-root-public", public, "-cert", certPath); err != nil {
		t.Errorf("verify-cert = %v, want ok", err)
	}
	later := func() time.Time { return testNow().AddDate(0, 0, 31) }
	if _, err := runTool(t, later, "verify-cert", "-root-public", public, "-cert", certPath); err == nil {
		t.Error("verify-cert accepted a certificate after its end")
	}
	other, err := runTool(t, testNow, "keygen", "-out", filepath.Join(dir, "other.key"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runTool(t, testNow, "verify-cert", "-root-public", other, "-cert", certPath); err == nil {
		t.Error("verify-cert accepted a certificate of another root")
	}
}

func TestSignCert_RefusesSerialZeroAndAShortEpochKey(t *testing.T) {
	t.Parallel()
	rootPath := filepath.Join(t.TempDir(), "root.key")
	if _, err := runTool(t, testNow, "keygen", "-out", rootPath); err != nil {
		t.Fatal(err)
	}

	if _, err := runTool(t, testNow, "sign-cert", "-root", rootPath, "-epoch-key", newEpochKeyHex(t), "-serial", "0", "-days", "30"); err == nil {
		t.Error("sign-cert accepted serial 0")
	}
	if _, err := runTool(t, testNow, "sign-cert", "-root", rootPath, "-epoch-key", "abcd", "-serial", "1", "-days", "30"); err == nil {
		t.Error("sign-cert accepted a short epoch key")
	}
}
