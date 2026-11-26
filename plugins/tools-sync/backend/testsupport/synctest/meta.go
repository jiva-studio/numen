package synctest

import (
	"encoding/json"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const metaContext = "numen-sync/meta/v1"

type Meta struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"`
	Hash  string `json:"hash"`
	Key   []byte `json:"key"`
	Kind  string `json:"kind"`
}

// OpenMeta opens enc_meta of the revision: 0x01, a nonce of 24 bytes and XChaCha20-Poly1305 of the JSON under
// the vault key, bound to the vault and the file.
func OpenMeta(t *testing.T, vaultKey [32]byte, revision *syncv1.Revision) Meta {
	t.Helper()
	sealed := revision.GetEncMeta()
	if len(sealed) < 1+chacha20poly1305.NonceSizeX || sealed[0] != 0x01 {
		t.Fatalf("enc_meta of %d bytes does not start with 0x01 and a nonce", len(sealed))
	}
	aead, err := chacha20poly1305.NewX(vaultKey[:])
	if err != nil {
		t.Fatal(err)
	}
	nonce, body := sealed[1:1+chacha20poly1305.NonceSizeX], sealed[1+chacha20poly1305.NonceSizeX:]
	associated := append([]byte(metaContext), revision.GetVaultId()...)
	associated = append(associated, revision.GetFileId()...)
	plain, err := aead.Open(nil, nonce, body, associated)
	if err != nil {
		t.Fatalf("enc_meta does not open with the vault key: %v", err)
	}
	var meta Meta
	if err := json.Unmarshal(plain, &meta); err != nil {
		t.Fatal(err)
	}
	return meta
}

func (c *Computer) OpenBlob(t *testing.T, vaultKey [32]byte, revision *syncv1.Revision) []byte {
	t.Helper()
	plain, isFound := c.FindBlob(t, vaultKey, revision)
	if !isFound {
		t.Fatal("the blob of the revision is not on the server")
	}
	return plain
}

func (c *Computer) FindBlob(t *testing.T, vaultKey [32]byte, revision *syncv1.Revision) ([]byte, bool) {
	t.Helper()
	meta := OpenMeta(t, vaultKey, revision)
	url, isFound := c.findBlobURL(t, revision)
	if !isFound {
		return nil, false
	}
	blob, err := c.Cloud.GetBlob(t.Context(), url)
	if err != nil {
		// A newer revision can replace the blob between the answer with its URL and the download.
		if _, isFound := c.findBlobURL(t, revision); !isFound {
			return nil, false
		}
		t.Fatal(err)
	}
	plain, err := blobcipher.Open([32]byte(meta.Key), [16]byte(revision.GetFileId()), blob)
	if err != nil {
		t.Fatalf("the blob does not open with the key of its metadata: %v", err)
	}
	return plain, true
}

func (c *Computer) findBlobURL(t *testing.T, revision *syncv1.Revision) (string, bool) {
	t.Helper()
	downloads, err := c.Cloud.GetBlobUrls(t.Context(), revision.GetVaultId(), [][]byte{revision.GetBlobId()})
	if err != nil {
		t.Fatal(err)
	}
	if len(downloads) != 1 {
		t.Fatalf("downloads = %v, want one", downloads)
	}
	return downloads[0].GetUrl(), downloads[0].GetFound()
}
