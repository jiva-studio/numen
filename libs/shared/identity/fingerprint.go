package identity

import (
	"crypto/ed25519"
	"crypto/sha256"
	"strings"
)

// Fingerprint is the first 16 bytes of SHA-256 over a label, a zero byte and a signing public key.
type Fingerprint [16]byte

func (f Fingerprint) String() string {
	const digits = "0123456789ABCDEF"
	var out strings.Builder
	for i, b := range f {
		if i > 0 && i%2 == 0 {
			out.WriteByte(' ')
		}
		out.WriteByte(digits[b>>4])
		out.WriteByte(digits[b&15])
	}
	return out.String()
}

func (p RecoveryPublicKey) Fingerprint() Fingerprint {
	return computeFingerprint(labelFingerprintAccount, p.Sign)
}

func (p DevicePublicKey) Fingerprint() Fingerprint {
	return computeFingerprint(labelFingerprintDevice, p.Sign)
}

func computeFingerprint(label string, sign ed25519.PublicKey) Fingerprint {
	sum := sha256.Sum256(append(append([]byte(label), 0), sign...))
	return Fingerprint(sum[:16])
}
