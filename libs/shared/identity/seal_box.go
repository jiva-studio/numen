package identity

import (
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/hpke"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	labelPluginBox = "numen/v1/plugin-box/"

	hpkeKEMX25519  = 0x0020
	hpkeKDFSHA256  = 0x0001
	hpkeAEADChaCha = 0x0003
	hpkeEncSize    = 32
)

// SealToAccountBox seals plain to the account box key of a person. Anyone may call it.
// The box is HPKE base mode (RFC 9180) with an ephemeral key read from random, so a fixed reader gives a fixed box.
func SealToAccountBox(random io.Reader, box [32]byte, pluginID string, context, plain []byte) ([]byte, error) {
	recipient, err := ecdh.X25519().NewPublicKey(box[:])
	if err != nil {
		return nil, fmt.Errorf("identity: recipient key: %w", err)
	}
	var seed [32]byte
	if _, err := io.ReadFull(random, seed[:]); err != nil {
		return nil, fmt.Errorf("identity: read ephemeral key: %w", err)
	}
	defer clear(seed[:])
	ephemeral, err := ecdh.X25519().NewPrivateKey(seed[:])
	if err != nil {
		return nil, fmt.Errorf("identity: ephemeral key: %w", err)
	}
	dh, err := ephemeral.ECDH(recipient)
	if err != nil {
		return nil, fmt.Errorf("identity: key agreement: %w", err)
	}
	enc := ephemeral.PublicKey().Bytes()
	sharedSecret, err := getBoxSharedSecret(dh, append(append([]byte(nil), enc...), box[:]...))
	if err != nil {
		return nil, err
	}
	key, nonce, err := getBoxKeySchedule(sharedSecret, getBoxInfo(pluginID, context))
	if err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("identity: box cipher: %w", err)
	}
	return aead.Seal(enc, nonce, plain, nil), nil
}

func (a AccountKeys) OpenSealedBox(pluginID string, context, sealed []byte) ([]byte, error) {
	if len(sealed) < hpkeEncSize+chacha20poly1305.Overhead {
		return nil, ErrUnwrap
	}
	private, err := ecdh.X25519().NewPrivateKey(a.box[:])
	if err != nil {
		return nil, ErrUnwrap
	}
	recipient, err := hpke.NewDHKEMPrivateKey(private)
	if err != nil {
		return nil, ErrUnwrap
	}
	plain, err := hpke.Open(recipient, hpke.HKDFSHA256(), hpke.ChaCha20Poly1305(), getBoxInfo(pluginID, context), sealed)
	if err != nil {
		return nil, ErrUnwrap
	}
	return plain, nil
}

func getBoxInfo(pluginID string, context []byte) []byte {
	info := make([]byte, 0, len(labelPluginBox)+len(pluginID)+1+len(context))
	info = append(info, labelPluginBox...)
	info = append(info, pluginID...)
	info = append(info, 0)
	return append(info, context...)
}

func getHPKESuite(kem bool) []byte {
	if kem {
		return binary.BigEndian.AppendUint16([]byte("KEM"), hpkeKEMX25519)
	}
	suite := binary.BigEndian.AppendUint16([]byte("HPKE"), hpkeKEMX25519)
	suite = binary.BigEndian.AppendUint16(suite, hpkeKDFSHA256)
	return binary.BigEndian.AppendUint16(suite, hpkeAEADChaCha)
}

func buildHPKELabel(suite []byte, label string, data []byte) []byte {
	labeled := append([]byte("HPKE-v1"), suite...)
	labeled = append(labeled, label...)
	return append(labeled, data...)
}

func extractHPKE(suite, salt []byte, label string, ikm []byte) ([]byte, error) {
	prk, err := hkdf.Extract(sha256.New, buildHPKELabel(suite, label, ikm), salt)
	if err != nil {
		return nil, fmt.Errorf("identity: box extract: %w", err)
	}
	return prk, nil
}

func expandHPKE(suite, prk []byte, label string, info []byte, size int) ([]byte, error) {
	labeled := binary.BigEndian.AppendUint16(nil, uint16(size))
	labeled = append(labeled, buildHPKELabel(suite, label, info)...)
	out, err := hkdf.Expand(sha256.New, prk, string(labeled), size)
	if err != nil {
		return nil, fmt.Errorf("identity: box expand: %w", err)
	}
	return out, nil
}

func getBoxSharedSecret(dh, kemContext []byte) ([]byte, error) {
	suite := getHPKESuite(true)
	prk, err := extractHPKE(suite, nil, "eae_prk", dh)
	if err != nil {
		return nil, err
	}
	return expandHPKE(suite, prk, "shared_secret", kemContext, sha256.Size)
}

func getBoxKeySchedule(sharedSecret, info []byte) (key, nonce []byte, err error) {
	suite := getHPKESuite(false)
	pskIDHash, err := extractHPKE(suite, nil, "psk_id_hash", nil)
	if err != nil {
		return nil, nil, err
	}
	infoHash, err := extractHPKE(suite, nil, "info_hash", info)
	if err != nil {
		return nil, nil, err
	}
	context := append([]byte{0}, pskIDHash...)
	context = append(context, infoHash...)
	secret, err := extractHPKE(suite, sharedSecret, "secret", nil)
	if err != nil {
		return nil, nil, err
	}
	if key, err = expandHPKE(suite, secret, "key", context, chacha20poly1305.KeySize); err != nil {
		return nil, nil, err
	}
	if nonce, err = expandHPKE(suite, secret, "base_nonce", context, chacha20poly1305.NonceSize); err != nil {
		return nil, nil, err
	}
	return key, nonce, nil
}
