package domain

import (
	"errors"
	"net/netip"
	"net/url"
	"strings"
)

// ErrNotAWebAddress is anything a link note cannot point at. Only `http` and
// `https` are fetched, and only away from this machine: neither a scheme
// reaching a file nor a host that is this machine is an address a note may
// carry.
var ErrNotAWebAddress = errors.New("not a web address")

// A WebAddress is where a link note points, in the one form two spellings of
// one resource share. What was fetched from it is kept under a name made from
// this form, so two notes that pasted the same video in different words hold
// one transcript between them.
type WebAddress struct {
	// URL is the address as it is fetched and as the note carries it.
	URL string
	// Video is the video this address names, and is empty for a page.
	Video string
}

// IsVideo reports whether there is a video at this address, which is what
// decides that a tab plays something and that words are asked for with times.
func (a WebAddress) IsVideo() bool { return a.Video != "" }

// embedded is where a frame plays a video from. The host serves no cookies of
// its own, and `enablejsapi` is what makes the frame answer the page holding
// it, so a passage is played from the second it was said without that host's
// script running inside the window.
const embedded = "https://www.youtube-nocookie.com/embed/"

// Embed is where a frame plays what is at this address, and nothing where
// nothing plays.
func (a WebAddress) Embed() string {
	if a.Video == "" {
		return ""
	}
	return embedded + a.Video + "?enablejsapi=1"
}

// EmbedHosts are the origins a window may frame. Every one of them runs its own
// scripts inside its own frame and reaches its own machines.
func EmbedHosts() []string { return []string{"https://www.youtube-nocookie.com"} }

// addressKey is what a link note writes where it points.
const addressKey = "url"

// ReadAddress is where a note carrying this frontmatter points, and what is
// wrong with what it wrote. It is read on a link note and nowhere else: an
// address on any other note is a key of the person's own.
//
// A link note with nowhere to point is missing its whole subject, so a missing
// address is a problem returned, and the note is read as every other note is.
func ReadAddress(front map[string]any) (WebAddress, []string) {
	raw, present := front[addressKey]
	if !present || raw == nil {
		return WebAddress{}, []string{"a link carries no " + addressKey}
	}
	written, isText := raw.(string)
	if !isText {
		return WebAddress{}, []string{addressKey + " is not text"}
	}
	at, err := ParseWebAddress(written)
	if err != nil {
		return WebAddress{}, []string{addressKey + " " + written + " is not a web address"}
	}
	return at, nil
}

// ParseWebAddress reads what a person pasted.
//
// The scheme and the host are lowercased, a default port and a fragment are
// dropped, and the parameters a site adds to say where a visitor came from are
// dropped with them; what is left is written in one order. A video is reduced to
// its identifier, so the share link, the watch page and the embed are one
// address.
func ParseWebAddress(raw string) (WebAddress, error) {
	address, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return WebAddress{}, ErrNotAWebAddress
	}
	address.Scheme = strings.ToLower(address.Scheme)
	if address.Scheme != "http" && address.Scheme != "https" {
		return WebAddress{}, ErrNotAWebAddress
	}
	address.Host = strings.ToLower(address.Host)
	if address.Hostname() == "" || thisMachine(address.Hostname()) {
		return WebAddress{}, ErrNotAWebAddress
	}
	if port := address.Port(); port == "80" && address.Scheme == "http" ||
		port == "443" && address.Scheme == "https" {
		address.Host = address.Hostname()
	}
	address.Fragment, address.RawFragment = "", ""
	address.User = nil

	if video := videoAt(address); video != "" {
		return WebAddress{URL: "https://www.youtube.com/watch?v=" + video, Video: video}, nil
	}
	address.RawQuery = kept(address.Query()).Encode()
	return WebAddress{URL: address.String()}, nil
}

// thisMachine says whether a host is this machine. A note points somewhere a
// browser would go, and what listens on this machine is not that: an index, a
// window's own socket and whatever else is running answer nobody's paste.
func thisMachine(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	address, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return false
	}
	return address.IsLoopback() || address.IsUnspecified()
}

// youtube are the hosts one video is published under. A host is written without
// `www.`, which is trimmed before the lookup.
var youtube = map[string]bool{
	"youtube.com":          true,
	"m.youtube.com":        true,
	"music.youtube.com":    true,
	"youtube-nocookie.com": true,
	"youtu.be":             true,
}

// videoAt is the video an address names, and nothing where it names none.
func videoAt(address *url.URL) string {
	host := strings.TrimPrefix(address.Hostname(), "www.")
	if !youtube[host] {
		return ""
	}
	if host == "youtu.be" {
		return videoID(strings.TrimPrefix(address.Path, "/"))
	}
	if address.Path == "/watch" {
		return videoID(address.Query().Get("v"))
	}
	for _, page := range []string{"/embed/", "/shorts/", "/live/", "/v/"} {
		if rest, found := strings.CutPrefix(address.Path, page); found {
			return videoID(rest)
		}
	}
	return ""
}

// videoID is the identifier where the segment is one, and nothing otherwise. It
// is eleven characters of the alphabet a URL carries unescaped, and a segment
// carrying anything else is a page of the site.
func videoID(segment string) string {
	segment, _, _ = strings.Cut(segment, "/")
	if len(segment) != 11 {
		return ""
	}
	for _, r := range segment {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return ""
		}
	}
	return segment
}

// campaign are the parameters that say where a visitor came from, beside every
// name beginning `utm_`. They are the same page whatever they hold, and a
// person who pastes one link from two places holds one note about it.
var campaign = map[string]bool{
	"fbclid":  true,
	"gclid":   true,
	"mc_cid":  true,
	"mc_eid":  true,
	"si":      true,
	"ref_src": true,
	"ref_url": true,
}

// kept is the query without them.
func kept(query url.Values) url.Values {
	for name := range query {
		folded := strings.ToLower(name)
		if campaign[folded] || strings.HasPrefix(folded, "utm_") {
			delete(query, name)
		}
	}
	return query
}
