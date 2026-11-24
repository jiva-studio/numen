package identity

import (
	"crypto/ed25519"
	"strings"
)

const recoverySignPrefix = "numen/v1/recovery-sign/"

func encodeRecoverySignMessage(pluginID string, message []byte) []byte {
	signed := make([]byte, 0, len(recoverySignPrefix)+len(pluginID)+1+len(message))
	signed = append(signed, recoverySignPrefix...)
	signed = append(signed, pluginID...)
	signed = append(signed, 0)
	return append(signed, message...)
}

// SignForRecovery signs message with the recovery signing key for one plugin, under a prefix no device signature shares.
func (k RecoveryKey) SignForRecovery(pluginID string, message []byte) [64]byte {
	return [64]byte(k.signEntry(encodeRecoverySignMessage(pluginID, message)))
}

// VerifyForRecovery checks a signature made by SignForRecovery; a plugin id with a zero byte never verifies.
func VerifyForRecovery(sign ed25519.PublicKey, pluginID string, message []byte, signature [64]byte) bool {
	if len(sign) != ed25519.PublicKeySize || strings.IndexByte(pluginID, 0) >= 0 {
		return false
	}
	return ed25519.Verify(sign, encodeRecoverySignMessage(pluginID, message), signature[:])
}
