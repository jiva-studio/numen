package domain

import (
	"errors"
	"net/url"
	"strings"
)

// ErrNotAWebAddress is anything a link note cannot point at. Only `http` and
// `https` are fetched: no scheme reaching a file or a socket on this machine is
// an address a note may carry.
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
	if address.Hostname() == "" {
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
// carrying anything else is a page of the site rather than a video on it.
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
