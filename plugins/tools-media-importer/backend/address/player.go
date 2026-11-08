package address

import (
	"net/url"
	"strings"
)

// PlayerOrigin is the origin the players of GetPlayerURL are framed from. The host sets no cookies
// of its own.
const PlayerOrigin = "https://www.youtube-nocookie.com"

func isPlayerSite(key string) bool {
	switch key {
	case "youtube.com", "m.youtube.com", "music.youtube.com", "youtube-nocookie.com", "youtu.be":
		return true
	}
	return false
}

// GetPlayerURL is the address of the player that plays the video at the address in a frame, and
// false where no player is known for it.
func GetPlayerURL(link URL) (string, bool) {
	parsed, err := url.Parse(string(link))
	if err != nil || !isPlayerSite(strings.TrimPrefix(parsed.Hostname(), "www.")) {
		return "", false
	}
	id := getVideoID(parsed)
	if id == "" {
		return "", false
	}
	return PlayerOrigin + "/embed/" + id, true
}

func getVideoID(parsed *url.URL) string {
	if strings.TrimPrefix(parsed.Hostname(), "www.") == "youtu.be" {
		return readVideoID(strings.TrimPrefix(parsed.Path, "/"))
	}
	if parsed.Path == "/watch" {
		return readVideoID(parsed.Query().Get("v"))
	}
	for _, page := range []string{"/embed/", "/shorts/", "/live/", "/v/"} {
		if rest, found := strings.CutPrefix(parsed.Path, page); found {
			return readVideoID(rest)
		}
	}
	return ""
}

func readVideoID(segment string) string {
	segment, _, _ = strings.Cut(segment, "/")
	if len(segment) != 11 {
		return ""
	}
	for _, r := range segment {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return ""
		}
	}
	return segment
}
