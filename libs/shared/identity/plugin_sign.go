package identity

import "crypto/ed25519"

const pluginSignPrefix = "numen/v1/plugin-sign/"

func pluginSignMessage(pluginID string, message []byte) []byte {
	signed := make([]byte, 0, len(pluginSignPrefix)+len(pluginID)+1+len(message))
	signed = append(signed, pluginSignPrefix...)
	signed = append(signed, pluginID...)
	signed = append(signed, 0)
	return append(signed, message...)
}

// SignForPlugin signs message with the device key under the plugin id.
func (d DeviceKey) SignForPlugin(pluginID string, message []byte) [64]byte {
	return [64]byte(d.signEntry(pluginSignMessage(pluginID, message)))
}

// VerifyForPlugin checks a signature made by SignForPlugin.
// ed25519.Verify rejects a non-canonical S.
func VerifyForPlugin(sign ed25519.PublicKey, pluginID string, message []byte, signature [64]byte) bool {
	if len(sign) != ed25519.PublicKeySize {
		return false
	}
	return ed25519.Verify(sign, pluginSignMessage(pluginID, message), signature[:])
}
