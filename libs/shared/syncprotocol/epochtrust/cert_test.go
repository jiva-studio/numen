package epochtrust_test

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func getNow() time.Time { return time.Unix(1_800_000_000, 0) }

func newKey(tb testing.TB, seed byte) (ed25519.PrivateKey, ed25519.PublicKey) {
	tb.Helper()
	private := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, ed25519.SeedSize))
	public, ok := private.Public().(ed25519.PublicKey)
	if !ok {
		tb.Fatal("not an Ed25519 key")
	}
	return private, public
}

func newCert(tb testing.TB, serial uint64, notBefore, notAfter time.Time) (epochtrust.Cert, ed25519.PublicKey, ed25519.PrivateKey) {
	tb.Helper()
	root, rootPublic := newKey(tb, 1)
	epochPrivate, epochPublic := newKey(tb, 2)
	cert, err := epochtrust.SignCert(root, epochPublic, serial, notBefore, notAfter)
	if err != nil {
		tb.Fatal(err)
	}
	return cert, rootPublic, epochPrivate
}

func TestVerifyCert_AcceptsACertificateOfTheRootInsideItsValidity(t *testing.T) {
	t.Parallel()
	cert, root, _ := newCert(t, 1, getNow().Add(-time.Hour), getNow().Add(time.Hour))

	if err := epochtrust.VerifyCert([]ed25519.PublicKey{root}, cert, getNow()); err != nil {
		t.Errorf("VerifyCert = %v, want nil", err)
	}
}

func TestVerifyCert_RefusesACertificateOutsideItsValidity(t *testing.T) {
	t.Parallel()
	cert, root, _ := newCert(t, 1, getNow().Add(-time.Hour), getNow().Add(time.Hour))
	for name, at := range map[string]time.Time{"before": getNow().Add(-2 * time.Hour), "at the end": getNow().Add(time.Hour), "after": getNow().Add(2 * time.Hour)} {
		if err := epochtrust.VerifyCert([]ed25519.PublicKey{root}, cert, at); !errors.Is(err, epochtrust.ErrUntrusted) {
			t.Errorf("%s: VerifyCert = %v, want ErrUntrusted", name, err)
		}
	}
}

func TestVerifyCert_RefusesACertificateOfAnotherRootAndAnEmptyRootList(t *testing.T) {
	t.Parallel()
	cert, _, _ := newCert(t, 1, getNow().Add(-time.Hour), getNow().Add(time.Hour))
	_, other := newKey(t, 9)

	if err := epochtrust.VerifyCert([]ed25519.PublicKey{other}, cert, getNow()); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("another root: VerifyCert = %v, want ErrUntrusted", err)
	}
	if err := epochtrust.VerifyCert(nil, cert, getNow()); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("no roots: VerifyCert = %v, want ErrUntrusted", err)
	}
}

func TestVerifyCert_RefusesACertificateWhoseFieldWasChanged(t *testing.T) {
	t.Parallel()
	cert, root, _ := newCert(t, 1, getNow().Add(-time.Hour), getNow().Add(time.Hour))
	cert.NotAfter += 3600

	if err := epochtrust.VerifyCert([]ed25519.PublicKey{root}, cert, getNow()); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("VerifyCert = %v, want ErrUntrusted", err)
	}
}

func TestSignCert_RefusesSerialZeroAndAnEmptyValidity(t *testing.T) {
	t.Parallel()
	root, _ := newKey(t, 1)
	_, epoch := newKey(t, 2)

	if _, err := epochtrust.SignCert(root, epoch, 0, getNow(), getNow().Add(time.Hour)); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("serial 0: SignCert = %v, want ErrUntrusted", err)
	}
	if _, err := epochtrust.SignCert(root, epoch, 1, getNow(), getNow()); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("empty validity: SignCert = %v, want ErrUntrusted", err)
	}
}

func TestVerifyCert_KeepsTheSerialInTheSignature(t *testing.T) {
	t.Parallel()
	cert, root, _ := newCert(t, 1, getNow().Add(-time.Hour), getNow().Add(time.Hour))
	cert.Serial = 2

	if err := epochtrust.VerifyCert([]ed25519.PublicKey{root}, cert, getNow()); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("VerifyCert = %v, want ErrUntrusted", err)
	}
}

func newAnnouncement(private ed25519.PrivateKey) *syncv1.RestoreEpoch {
	id := bytes.Repeat([]byte{0x31}, 16)
	return &syncv1.RestoreEpoch{Id: id, IssuedAt: getNow().Unix(), Signature: ed25519.Sign(private, syncprotocol.GetRestoreEpochMessage(id, getNow().Unix()))}
}

func TestVerifyAnnouncement_AcceptsTheAnnouncementOfTheCertifiedKey(t *testing.T) {
	t.Parallel()
	cert, _, private := newCert(t, 1, getNow().Add(-time.Hour), getNow().Add(time.Hour))

	if err := epochtrust.VerifyAnnouncement(cert, newAnnouncement(private)); err != nil {
		t.Errorf("VerifyAnnouncement = %v, want nil", err)
	}
}

func TestVerifyAnnouncement_RefusesTheAnnouncementOfAnotherKey(t *testing.T) {
	t.Parallel()
	cert, _, _ := newCert(t, 1, getNow().Add(-time.Hour), getNow().Add(time.Hour))
	other, _ := newKey(t, 9)

	if err := epochtrust.VerifyAnnouncement(cert, newAnnouncement(other)); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("VerifyAnnouncement = %v, want ErrUntrusted", err)
	}
	if err := epochtrust.VerifyAnnouncement(cert, nil); !errors.Is(err, epochtrust.ErrUntrusted) {
		t.Errorf("no announcement: VerifyAnnouncement = %v, want ErrUntrusted", err)
	}
}

func TestGetRoots_ReturnsTheEmbeddedRootForTheDefaultServer(t *testing.T) {
	t.Parallel()
	want, err := hex.DecodeString(epochtrust.EmbeddedRootHex)
	if err != nil || len(want) != ed25519.PublicKeySize {
		t.Fatalf("EmbeddedRootHex decodes to %d bytes, error %v, want 32", len(want), err)
	}

	roots, err := epochtrust.GetRoots("https://sync.numen.md", "")

	if err != nil || len(roots) != 1 || !bytes.Equal(roots[0], want) {
		t.Errorf("GetRoots = %x %v, want the embedded root", roots, err)
	}
}

func TestGetRoots_HasNoRootForAnotherServerWithoutASetting(t *testing.T) {
	t.Parallel()

	roots, err := epochtrust.GetRoots("http://127.0.0.1:8080", "")

	if err != nil || len(roots) != 0 {
		t.Errorf("GetRoots = %x %v, want none", roots, err)
	}
}

func TestGetRoots_AddsTheRootOfTheSettingForALoopbackServer(t *testing.T) {
	t.Parallel()
	_, public := newKey(t, 5)

	roots, err := epochtrust.GetRoots("http://127.0.0.1:8080", hexOf(public))

	if err != nil || len(roots) != 1 || !bytes.Equal(roots[0], public) {
		t.Errorf("GetRoots = %x %v, want the key of the setting", roots, err)
	}
}

func TestGetRoots_NeverAddsTheRootOfTheSettingForTheDefaultServer(t *testing.T) {
	t.Parallel()
	_, public := newKey(t, 5)

	for _, address := range []string{epochtrust.DefaultServer, epochtrust.DefaultServer + "/"} {
		roots, err := epochtrust.GetRoots(address, hexOf(public))
		if err != nil || len(roots) != 1 || hexOf(roots[0]) != epochtrust.EmbeddedRootHex {
			t.Errorf("%s: GetRoots = %x %v, want only the embedded root", address, roots, err)
		}
	}
}

func TestGetRoots_RefusesARootThatIsNotThirtyTwoBytesOfHex(t *testing.T) {
	t.Parallel()

	roots, err := epochtrust.GetRoots("http://127.0.0.1:8080", "abcd")

	if err == nil || len(roots) != 0 {
		t.Errorf("GetRoots = %x %v, want an error and no root", roots, err)
	}
}

func hexOf(key []byte) string { return hex.EncodeToString(key) }

func TestGetRoots_IgnoresTheRootOfTheSettingForEverySpellingOfTheDefaultServer(t *testing.T) {
	t.Parallel()
	public, _ := newKey(t, 7)
	for _, address := range []string{
		"https://sync.numen.md",
		"https://sync.numen.md/",
		"https://sync.numen.md:443",
		"https://sync.numen.md.",
		"HTTPS://SYNC.NUMEN.MD/",
		"https://sync.numen.md/some/path",
	} {
		roots, err := epochtrust.GetRoots(address, hexOf(public))
		if err != nil || len(roots) != 1 || hexOf(roots[0]) != epochtrust.EmbeddedRootHex {
			t.Errorf("GetRoots(%q) = %x %v, want only the embedded root", address, roots, err)
		}
	}
}

func TestGetRoots_ReturnsOnlyTheRootOfTheSettingForAnotherServer(t *testing.T) {
	t.Parallel()
	public, _ := newKey(t, 7)
	for _, address := range []string{"https://sync.example.com", "https://sync.numen.md:8443", "http://sync.numen.md", "https://evilsync.numen.md"} {
		roots, err := epochtrust.GetRoots(address, hexOf(public))
		if err != nil || len(roots) != 1 || !bytes.Equal(roots[0], public) {
			t.Errorf("GetRoots(%q) = %x %v, want the key of the setting", address, roots, err)
		}
	}
}
