// Package epochtrust is how a client decides to believe a restore epoch: an offline root key of the operator signs a
// certificate for the epoch key of the server, and the epoch key signs the announcement. A server alone cannot
// make a restore announcement a client believes.
package epochtrust

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const certLabel = "numen-sync/epoch-cert/v1"

// ErrUntrusted means a certificate or an announcement is missing, malformed, expired, or not signed by a root key
// this client knows.
var ErrUntrusted = errors.New("the restore epoch of the server is not trusted")

// Cert is the delegation of the root key to an epoch key. Times are unix seconds.
type Cert struct {
	EpochKey      []byte
	Serial        uint64
	NotBefore     int64
	NotAfter      int64
	RootSignature []byte
}

// GetCertMessage returns the bytes the root key signs: the label, a zero byte, the epoch key, then serial,
// not-before and not-after as 8 bytes big endian each.
func GetCertMessage(cert Cert) []byte {
	message := append([]byte(certLabel), 0)
	message = append(message, cert.EpochKey...)
	message = binary.BigEndian.AppendUint64(message, cert.Serial)
	message = binary.BigEndian.AppendUint64(message, uint64(cert.NotBefore))
	return binary.BigEndian.AppendUint64(message, uint64(cert.NotAfter))
}

// SignCert returns the certificate of epochKey signed by the root key. The serial starts at 1.
func SignCert(root ed25519.PrivateKey, epochKey ed25519.PublicKey, serial uint64, notBefore, notAfter time.Time) (Cert, error) {
	cert := Cert{EpochKey: epochKey, Serial: serial, NotBefore: notBefore.Unix(), NotAfter: notAfter.Unix(), RootSignature: nil}
	if len(root) != ed25519.PrivateKeySize || len(epochKey) != ed25519.PublicKeySize {
		return Cert{}, fmt.Errorf("%w: the keys have the wrong size", ErrUntrusted)
	}
	if err := checkShape(cert); err != nil {
		return Cert{}, err
	}
	cert.RootSignature = ed25519.Sign(root, GetCertMessage(cert))
	return cert, nil
}

func checkShape(cert Cert) error {
	switch {
	case cert.Serial == 0:
		return fmt.Errorf("%w: the serial is 0", ErrUntrusted)
	case cert.NotAfter <= cert.NotBefore:
		return fmt.Errorf("%w: the validity is empty", ErrUntrusted)
	}
	return nil
}

// VerifyCert checks that one of the roots signed the certificate and that now is inside its validity. With no roots
// every certificate is refused.
func VerifyCert(roots []ed25519.PublicKey, cert Cert, now time.Time) error {
	if len(cert.EpochKey) != ed25519.PublicKeySize || len(cert.RootSignature) != ed25519.SignatureSize {
		return fmt.Errorf("%w: the certificate is missing or has the wrong size", ErrUntrusted)
	}
	if err := checkShape(cert); err != nil {
		return err
	}
	if now.Unix() < cert.NotBefore || now.Unix() >= cert.NotAfter {
		return fmt.Errorf("%w: the certificate is not valid now", ErrUntrusted)
	}
	message := GetCertMessage(cert)
	for _, root := range roots {
		if len(root) == ed25519.PublicKeySize && ed25519.Verify(root, message, cert.RootSignature) {
			return nil
		}
	}
	return fmt.Errorf("%w: no known root key signed the certificate", ErrUntrusted)
}

// VerifyAnnouncement checks the announcement against the epoch key of a certificate that VerifyCert accepted.
func VerifyAnnouncement(cert Cert, epoch *syncv1.RestoreEpoch) error {
	if err := syncprotocol.CheckRestoreEpoch(cert.EpochKey, epoch); err != nil {
		return fmt.Errorf("%w: %w", ErrUntrusted, err)
	}
	return nil
}
