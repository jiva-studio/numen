package searchrun

import (
	"encoding/base64"
	"strconv"
	"strings"
)

// EncodePageToken returns the token of the results of a search from offset on.
func EncodePageToken(id ID, offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(string(id) + ":" + strconv.Itoa(offset)))
}

// ParsePageToken returns the offset a token names in the results of the search id, which holds total
// results. A token of another search, an offset outside the results or text that is no token returns
// ErrInvalidPageToken.
func ParsePageToken(token string, id ID, total int) (int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, ErrInvalidPageToken
	}
	named, offsetText, isCut := strings.Cut(string(decoded), ":")
	if !isCut || named != string(id) {
		return 0, ErrInvalidPageToken
	}
	offset, err := strconv.Atoi(offsetText)
	if err != nil || offset < 0 || offset > total {
		return 0, ErrInvalidPageToken
	}
	return offset, nil
}
