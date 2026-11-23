package client_test

import (
	"bytes"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

func newRevision() *syncv1.Revision {
	return &syncv1.Revision{
		VaultId:             bytes.Repeat([]byte{1}, 16),
		FileId:              bytes.Repeat([]byte{2}, 16),
		ParentRev:           bytes.Repeat([]byte{3}, 32),
		BlobId:              bytes.Repeat([]byte{4}, 16),
		BlobHash:            bytes.Repeat([]byte{5}, 32),
		BlobSize:            4096,
		EncMeta:             []byte("meta"),
		PathHash:            bytes.Repeat([]byte{6}, 32),
		AuthorDeviceSignKey: bytes.Repeat([]byte{7}, 32),
		Signature:           bytes.Repeat([]byte{8}, 64),
	}
}

func newCatalogEntry() *syncv1.CatalogEntry {
	return &syncv1.CatalogEntry{
		Version:             3,
		PrevHash:            bytes.Repeat([]byte{9}, 32),
		VaultId:             bytes.Repeat([]byte{1}, 16),
		CreatesVault:        true,
		EncBody:             []byte("body"),
		AuthorDeviceSignKey: bytes.Repeat([]byte{7}, 32),
		Signature:           bytes.Repeat([]byte{8}, 64),
	}
}
