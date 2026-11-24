package share

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

const (
	linkPrefix   = "numen-share:"
	linkVersion  = 1
	checkSize    = 4
	maxLinkSize  = 2048
	maxNameBytes = 120
)

// LinkInvitation is what the link a folder is shared by tells the person who pastes it: the one-time handshake key of
// the sharer with the address of the relay, the name of the folder, the fingerprint of the account of the sharer and
// the moment the link stops working.
type LinkInvitation struct {
	Payload     identity.LinkPayload
	Folder      string
	Fingerprint string
	ExpiresAt   time.Time
}

type linkBody struct {
	Version     int    `json:"v"`
	Payload     string `json:"p"`
	Folder      string `json:"f"`
	Fingerprint string `json:"i"`
	ExpiresAt   int64  `json:"e"`
}

// NewLink returns the text of the link: a prefix, then the base64url of the body and four bytes of its digest, which
// catch a link that was cut or mistyped.
func NewLink(invitation LinkInvitation) (string, error) {
	raw, err := json.Marshal(linkBody{
		Version: linkVersion, Payload: invitation.Payload.Text(), Folder: CleanName(invitation.Folder),
		Fingerprint: invitation.Fingerprint, ExpiresAt: invitation.ExpiresAt.Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("write the link: %w", err)
	}
	digest := sha256.Sum256(raw)
	return linkPrefix + base64.RawURLEncoding.EncodeToString(append(raw, digest[:checkSize]...)), nil
}

// ParseLink returns the invitation a link holds. A text that is not a link of this version, whose digest does not match
// or whose key, channel or server is malformed is ErrBadLink.
func ParseLink(text string) (LinkInvitation, error) {
	encoded, ok := strings.CutPrefix(strings.TrimSpace(text), linkPrefix)
	if !ok || len(encoded) > maxLinkSize {
		return LinkInvitation{}, ErrBadLink
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) <= checkSize {
		return LinkInvitation{}, ErrBadLink
	}
	body, check := raw[:len(raw)-checkSize], raw[len(raw)-checkSize:]
	digest := sha256.Sum256(body)
	if string(digest[:checkSize]) != string(check) {
		return LinkInvitation{}, ErrBadLink
	}
	var parsed linkBody
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Version != linkVersion {
		return LinkInvitation{}, ErrBadLink
	}
	payload, err := identity.ParseLinkPayload(parsed.Payload)
	if err != nil || payload.Server == "" {
		return LinkInvitation{}, ErrBadLink
	}
	return LinkInvitation{
		Payload: payload, Folder: CleanName(parsed.Folder), Fingerprint: parsed.Fingerprint, ExpiresAt: time.Unix(parsed.ExpiresAt, 0).UTC(),
	}, nil
}

// CleanDeviceName returns the name a device announces, shortened to what a request shows and never empty.
func CleanDeviceName(name string) string {
	cleaned := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < ' ' {
			return -1
		}
		return r
	}, name))
	if len(cleaned) > maxNameBytes {
		cleaned = strings.ToValidUTF8(cleaned[:maxNameBytes], "")
	}
	if cleaned == "" {
		return "Unknown device"
	}
	return cleaned
}
