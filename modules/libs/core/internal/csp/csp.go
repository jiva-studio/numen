// Package csp is the Content-Security-Policy a window is served under: what it
// may load, and from where.
//
// It is not how a window looks. A policy is what a browser refuses on a page's
// behalf, and widening it is a decision about what may run in front of a
// person's vault.
package csp

import "strings"

// Policy is what a window may load: what its own handler serves, and for a
// picture, a sound or a frame whatever Sources names beside it. No script runs
// that it did not serve, no handler written in an attribute runs at all, and no
// form is submitted anywhere.
//
// Where the window itself goes is no directive of a policy. A link leading
// outward is held by the page.
//
// Inline style is allowed: a page positions what it draws through the style
// attribute.
func (s Sources) Policy() string {
	return "default-src 'self'; img-src " + listSources(s.Images) +
		"; media-src " + listSources(s.Media) +
		"; frame-src " + listSources(s.Frames) +
		"; style-src 'self' 'unsafe-inline'; " +
		"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
		"form-action 'none'; frame-ancestors 'none'"
}

// Sources are the places a window may load from besides what its own handler
// serves. One field to a directive of the policy: widening where a picture
// comes from must not widen where sound does.
type Sources struct {
	// Images fills img-src. A card carries its own bytes, and a `data:` URI is
	// no request.
	Images []string
	// Media fills media-src, which is where sound and video are loaded from. A
	// browser fetches those down a path of its own, which speaks the protocols
	// of the world and not the scheme a window is drawn from.
	Media []string
	// Frames fills frame-src, which is where a page a link note points at is
	// played from. Every host here runs its own scripts inside its own frame
	// and reaches its own machines, and the window is what names them.
	Frames []string
}

// listSources is a window's own handler and whatever else is allowed beside it.
func listSources(also []string) string {
	return strings.Join(append([]string{"'self'"}, also...), " ")
}
