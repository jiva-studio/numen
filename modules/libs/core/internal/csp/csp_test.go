package csp_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/csp"
)

// Every window is held to one policy. A theme is an ordinary stylesheet a
// person may have downloaded, and this is the whole of what stops one reaching
// the network.
func TestAWindowIsHeldToOnePolicy(t *testing.T) {
	const held = "default-src 'self'; img-src 'self'; media-src 'self'; frame-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
		"form-action 'none'; frame-ancestors 'none'"
	if got := (csp.Sources{}).Policy(); got != held {
		t.Errorf("the policy reads %q", got)
	}
}

// A window drawing text that carries its own pictures says where a picture may
// come from, and changes nothing else.
func TestAWindowDrawingItsOwnPicturesSaysSoAndNoMore(t *testing.T) {
	own := csp.Sources{}.Policy()
	with := csp.Sources{Images: []string{"data:"}}.Policy()

	if with == own {
		t.Fatal("naming where pictures come from changed nothing")
	}
	if !strings.Contains(with, "img-src 'self' data:;") {
		t.Errorf("the policy reads %q", with)
	}
	// Everything the two have in common is every directive but that one.
	for _, directive := range strings.Split(own, "; ") {
		if strings.HasPrefix(directive, "img-src") {
			continue
		}
		if !strings.Contains(with, directive) {
			t.Errorf("%q is not in %q", directive, with)
		}
	}
}

// A form is submitted nowhere. A page that carries text from another person can
// carry a form with it, and `form-action` is what a policy says about one:
// nothing else in the policy answers for where a form posts to.
func TestAFormIsSubmittedNowhere(t *testing.T) {
	if !strings.Contains(csp.Sources{}.Policy(), "form-action 'none'") {
		t.Error("a form on the page could be submitted somewhere")
	}
}
