package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"fmt"
	"io"
)

const (
	sealVersion       = 0x01
	sealSaltSize      = 16
	sealKeySize       = 32
	sealNonceSize     = 12
	labelPluginKey    = "numen/v1/plugin/"
	labelSealKey      = "numen/v1/seal"
	labelPluginSealAD = "numen/v1/plugin-seal/"
)

func (a AccountKeys) getPluginKey(pluginID string) ([]byte, error) {
	return hkdf.Key(sha256.New, a.kek[:], nil, labelPluginKey+pluginID, sealKeySize)
}

func newSealAEAD(pluginKey, salt []byte) (cipher.AEAD, error) {
	subkey, err := hkdf.Key(sha256.New, pluginKey, salt, labelSealKey, sealKeySize)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(subkey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func getSealAssociatedData(pluginID string, context []byte) []byte {
	data := make([]byte, 0, len(labelPluginSealAD)+len(pluginID)+1+len(context))
	data = append(data, labelPluginSealAD...)
	data = append(data, pluginID...)
	data = append(data, 0)
	return append(data, context...)
}

// SealForPlugin encrypts plain under a key only this plugin can derive from the account, bound to context.
// The subkey is unique per random salt, so the nonce is fixed.
func (a AccountKeys) SealForPlugin(random io.Reader, pluginID string, context, plain []byte) ([]byte, error) {
	salt := make([]byte, sealSaltSize)
	if _, err := io.ReadFull(random, salt); err != nil {
		return nil, fmt.Errorf("identity: read seal salt: %w", err)
	}
	pluginKey, err := a.getPluginKey(pluginID)
	if err != nil {
		return nil, fmt.Errorf("identity: derive plugin key: %w", err)
	}
	aead, err := newSealAEAD(pluginKey, salt)
	if err != nil {
		return nil, fmt.Errorf("identity: derive seal key: %w", err)
	}
	sealed := append([]byte{sealVersion}, salt...)
	return aead.Seal(sealed, make([]byte, sealNonceSize), plain, getSealAssociatedData(pluginID, context)), nil
}

// OpenForPlugin decrypts what SealForPlugin sealed for the same plugin and context. Any failure is ErrUnwrap.
func (a AccountKeys) OpenForPlugin(pluginID string, context, sealed []byte) ([]byte, error) {
	if len(sealed) < 1+sealSaltSize || sealed[0] != sealVersion {
		return nil, ErrUnwrap
	}
	pluginKey, err := a.getPluginKey(pluginID)
	if err != nil {
		return nil, ErrUnwrap
	}
	aead, err := newSealAEAD(pluginKey, sealed[1:1+sealSaltSize])
	if err != nil {
		return nil, ErrUnwrap
	}
	plain, err := aead.Open(nil, make([]byte, sealNonceSize), sealed[1+sealSaltSize:], getSealAssociatedData(pluginID, context))
	if err != nil {
		return nil, ErrUnwrap
	}
	return plain, nil
}
