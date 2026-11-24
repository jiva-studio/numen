// Package persisted is the JSON form of the parts of the server that Snapshot keeps and Restore reads.
package persisted

import (
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver/internal/sharestate"
)

// Multipart is the state of one multipart upload.
type Multipart struct {
	PartSize    uint64            `json:"partSize"`
	PartCount   uint32            `json:"partCount"`
	Parts       map[uint32]uint64 `json:"parts"`
	IsCompleted bool              `json:"isCompleted"`
	IsAborted   bool              `json:"isAborted"`
}

// Share is the state of one share.
type Share struct {
	Account  string               `json:"account"`
	Vault    string               `json:"vault"`
	Epoch    uint32               `json:"epoch"`
	IsClosed bool                 `json:"isClosed"`
	Log      [][]byte             `json:"log"`
	Members  []*sharestate.Member `json:"members"`
	Exits    map[string]uint64    `json:"exits"`
}

// Vault is the state of one vault.
type Vault struct {
	Account   string            `json:"account"`
	Seq       uint64            `json:"seq"`
	PrunedSeq uint64            `json:"prunedSeq"`
	Files     map[string][]byte `json:"files"`
	Paths     map[string]string `json:"paths"`
}

// Blob is the state of one blob.
type Blob struct {
	Account    string     `json:"account"`
	Vault      string     `json:"vault"`
	Size       uint64     `json:"size"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	State      int        `json:"state"`
	IsUploaded bool       `json:"isUploaded"`
	Multipart  *Multipart `json:"multipart"`
}
