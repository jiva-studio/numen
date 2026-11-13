package model

import "testing"

func TestThePrefixesOfAConnectionProfileChangeTheHandlingHash(t *testing.T) {
	plain := ConnectionProfile{Model: "m", MaxTokens: 512}
	prefixed := plain
	prefixed.QueryPrefix, prefixed.DocumentPrefix = "query: ", "passage: "

	if plain.GetHandlingHash() != "" {
		t.Errorf("handling hash without prefixes = %q, want none", plain.GetHandlingHash())
	}
	if prefixed.GetHandlingHash() == "" {
		t.Error("handling hash with prefixes is empty")
	}
}

func TestTheHandlingOfAConnectionProfileAppliesItsPrefixToTheRole(t *testing.T) {
	profile := ConnectionProfile{QueryPrefix: "query: ", DocumentPrefix: "passage: "}

	if got := profile.GetHandling().Apply(RoleQuery, []string{"a"}); got[0] != "query: a" {
		t.Errorf("query = %q", got[0])
	}
	if got := profile.GetHandling().Apply(RoleDocument, []string{"a"}); got[0] != "passage: a" {
		t.Errorf("document = %q", got[0])
	}
}
