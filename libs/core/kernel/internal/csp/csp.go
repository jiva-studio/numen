// Package csp is the Content-Security-Policy a window is served under: what it
// may load, and from where.
//
// It is not about appearance. A policy is what a browser blocks on a page, and
// widening it is a decision about what may run in front of the user's vault.
package csp

import "strings"

// Policy is what a window may load: what its own handler serves, and for a
// picture, a sound or a frame whatever Sources names beside it.
// No script runs that it did not serve, no attribute handler runs, no form is submitted.
// Where the window itself goes is no directive; the page holds outward links.
// Inline style is allowed: a page positions what it draws through the style attribute.
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

// GetPluginPagePolicy is what a page of a plugin's store may load: its own files and the frames named. It
// runs scripts of its own files, which the window's policy forbids.
func GetPluginPagePolicy(frames []string) string {
	return "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
		"frame-src " + listSources(frames) +
		"; object-src 'none'; base-uri 'none'; form-action 'none'"
}

// GetPictureOnlyPolicy is what a file served as a picture may load: nothing. An SVG opened as a page runs
// its scripts; under this policy it does not.
func GetPictureOnlyPolicy() string {
	return "default-src 'none'; style-src 'unsafe-inline'; sandbox"
}

// PermissionsPolicy lets the page of a window use the camera and no frame use it, and gives no page the microphone.
const PermissionsPolicy = "camera=(self), microphone=()"

// EditorPermissionsPolicy lets the page of the editor window use the camera and the microphone, and no frame use either.
const EditorPermissionsPolicy = "camera=(self), microphone=(self)"
