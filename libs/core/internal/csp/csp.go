// Package csp is the Content-Security-Policy a window is served under: what it
// may load, and from where.
//
// It is not about appearance. A policy is what a browser blocks on a page, and
// widening it is a decision about what may run in front of the user's vault.
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
		"font-src 'self'; connect-src " + listSources(s.Connect) + "; object-src 'none'; base-uri 'none'; " +
		"form-action 'none'; frame-ancestors 'none'"
}

// Sources are the places a window may load from besides what its own handler
// serves. One field to a directive of the policy: widening where a picture
// comes from must not widen where sound does.
type Sources struct {
	Images  []string
	Media   []string
	Frames  []string
	Connect []string
}

func listSources(also []string) string {
	return strings.Join(append([]string{"'self'"}, also...), " ")
}
