package domain

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// ErrInvalidContentType is text that is no MIME type and no mask of one.
var ErrInvalidContentType = errors.New("invalid content type")

// ContentType is a MIME type, such as `application/pdf`, or a mask of them: `audio/*` is every
// audio type and `*/*` every type. It is held in lowercase, as MIME types compare without regard
// to case.
type ContentType string

// UnknownContentType is the type of a file whose name matches no table entry (RFC 2046 §4.5.1).
const UnknownContentType ContentType = "application/octet-stream"

// AnyContentType is the mask of every type, a file of an unknown type among them.
const AnyContentType ContentType = "*/*"

// RFC 6838 §4.2 restricted-name characters.
var contentTypePattern = regexp.MustCompile(`^([a-z0-9][a-z0-9!#$&^_.+-]{0,126})/([a-z0-9][a-z0-9!#$&^_.+-]{0,126}|\*)$`)

// ParseContentType reads a MIME type or a mask of one.
func ParseContentType(text string) (ContentType, error) {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == string(AnyContentType) || contentTypePattern.MatchString(lower) {
		return ContentType(lower), nil
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidContentType, text)
}

// IsMask reports whether the type matches more than one type.
func (t ContentType) IsMask() bool {
	return strings.HasSuffix(string(t), "/*")
}

// Matches reports whether the type is the given one or a mask that covers it.
func (t ContentType) Matches(other ContentType) bool {
	switch {
	case t == AnyContentType:
		return true
	case t.IsMask():
		return strings.HasPrefix(string(other), strings.TrimSuffix(string(t), "*")) && !other.IsMask()
	}
	return t == other
}

// getWidth orders types from the narrowest: a type, a mask of one top-level type, every type.
func (t ContentType) getWidth() int {
	switch {
	case t == AnyContentType:
		return 2
	case t.IsMask():
		return 1
	}
	return 0
}

// HandlerRank is the priority of a handler among the handlers of a type: a default handler is chosen
// without prompting, an alternate handler only on explicit request.
type HandlerRank string

const (
	RankDefault   HandlerRank = "default"
	RankAlternate HandlerRank = "alternate"
)

// IsValid reports whether the rank is one of the two.
func (r HandlerRank) IsValid() bool {
	return r == RankDefault || r == RankAlternate
}

// ContentHandler is one way a plugin declares to handle files of a type.
type ContentHandler struct {
	PluginID string
	// ID names the handler among those of its plugin.
	ID   string
	Type ContentType
	Rank HandlerRank
}

// ChooseHandler is the default handler of a file of the type, and false where none of the candidates
// is a default one for it. Among several, the one of the narrower type wins, then the one of the
// lower plugin id, then of the lower handler id, so the choice never depends on their order.
func ChooseHandler(of ContentType, candidates []ContentHandler) (ContentHandler, bool) {
	var matching []ContentHandler
	for _, one := range candidates {
		if one.Rank == RankDefault && one.Type.Matches(of) {
			matching = append(matching, one)
		}
	}
	if len(matching) == 0 {
		return ContentHandler{}, false
	}
	return slices.MinFunc(matching, func(a, b ContentHandler) int {
		return cmp.Or(
			cmp.Compare(a.Type.getWidth(), b.Type.getWidth()),
			strings.Compare(a.PluginID, b.PluginID),
			strings.Compare(a.ID, b.ID),
		)
	}), true
}
