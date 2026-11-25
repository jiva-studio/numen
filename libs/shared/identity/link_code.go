package identity

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const labelLinkAccountsCode = "numen/v1/link/accounts-code"

func newLinkCodeKey(pseudo []byte, transcript [32]byte) ([32]byte, error) {
	raw, err := hkdf.Expand(sha256.New, pseudo, labelLinkAccountsCode+string(transcript[:]), sha256.Size)
	if err != nil {
		return [32]byte{}, err
	}
	return [32]byte(raw), nil
}

// GetCodeFor returns the six digits both sides derive from the keys of the link and the fingerprints of the accounts that meet on it, given
// in the same order on both sides. Whoever holds the link alone, or holds it with another account, shows another code than the sides
// that name the two accounts.
func (c *LinkChannel) GetCodeFor(fingerprints ...string) string {
	mac := hmac.New(sha256.New, c.codeKey[:])
	for _, fingerprint := range fingerprints {
		mac.Write(binary.BigEndian.AppendUint32(nil, uint32(len(fingerprint))))
		mac.Write([]byte(fingerprint))
	}
	return fmt.Sprintf("%06d", binary.BigEndian.Uint64(mac.Sum(nil)[:8])%linkCodeDigits)
}
