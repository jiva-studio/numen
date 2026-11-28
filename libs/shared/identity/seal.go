package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	sealVersion       = 0x02
	sealHeaderSize    = 1 + 4
	sealSaltSize      = 16
	sealKeySize       = 32
	sealNonceSize     = 12
	labelPluginKey    = "numen/v1/plugin/"
	labelSealKey      = "numen/v1/seal"
	labelPluginSealAD = "numen/v1/plugin-seal/"
)

func (a AccountKeys) getKEK(epoch uint32) ([32]byte, bool) {
	if epoch == a.epoch {
		return a.kek, true
	}
	kek, ok := a.earlier[epoch]
	return kek, ok
}

func getPluginKey(kek [32]byte, pluginID string) ([]byte, error) {
	return hkdf.Key(sha256.New, kek[:], nil, labelPluginKey+pluginID, sealKeySize)
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

func getSealAssociatedData(pluginID string, epoch uint32, context []byte) []byte {
	data := make([]byte, 0, len(labelPluginSealAD)+len(pluginID)+1+4+len(context))
	data = append(data, labelPluginSealAD...)
	data = append(data, pluginID...)
	data = append(data, 0)
	data = binary.BigEndian.AppendUint32(data, epoch)
	return append(data, context...)
}

// SealForPlugin encrypts plain under a key only this plugin can derive from the account keys of the current epoch, bound to context.
// The sealed bytes name the epoch. The subkey is unique per random salt, so the nonce is fixed.
func (a AccountKeys) SealForPlugin(random io.Reader, pluginID string, context, plain []byte) ([]byte, error) {
	salt := make([]byte, sealSaltSize)
	if _, err := io.ReadFull(random, salt); err != nil {
		return nil, fmt.Errorf("identity: read seal salt: %w", err)
	}
	pluginKey, err := getPluginKey(a.kek, pluginID)
	if err != nil {
		return nil, fmt.Errorf("identity: derive plugin key: %w", err)
	}
	aead, err := newSealAEAD(pluginKey, salt)
	if err != nil {
		return nil, fmt.Errorf("identity: derive seal key: %w", err)
	}
	sealed := binary.BigEndian.AppendUint32([]byte{sealVersion}, a.epoch)
	sealed = append(sealed, salt...)
	return aead.Seal(sealed, make([]byte, sealNonceSize), plain, getSealAssociatedData(pluginID, a.epoch, context)), nil
}

// OpenForPlugin decrypts what SealForPlugin sealed for the same plugin and context, under any epoch the keys hold.
// Any failure, an epoch the keys do not hold included, is ErrUnwrap.
func (a AccountKeys) OpenForPlugin(pluginID string, context, sealed []byte) ([]byte, error) {
	if len(sealed) < sealHeaderSize+sealSaltSize || sealed[0] != sealVersion {
		return nil, ErrUnwrap
	}
	epoch := binary.BigEndian.Uint32(sealed[1:sealHeaderSize])
	kek, ok := a.getKEK(epoch)
	if !ok {
		return nil, ErrUnwrap
	}
	pluginKey, err := getPluginKey(kek, pluginID)
	if err != nil {
		return nil, ErrUnwrap
	}
	salt := sealed[sealHeaderSize : sealHeaderSize+sealSaltSize]
	aead, err := newSealAEAD(pluginKey, salt)
	if err != nil {
		return nil, ErrUnwrap
	}
	plain, err := aead.Open(nil, make([]byte, sealNonceSize), sealed[sealHeaderSize+sealSaltSize:], getSealAssociatedData(pluginID, epoch, context))
	if err != nil {
		return nil, ErrUnwrap
	}
	return plain, nil
}
