package manifestjson_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readSecrets(secrets string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{
		"id": "a.b", "name": "A", "version": "1.0.0", "backend": {"entrypoint": "bin/a"},
		"settings": [
			{"key": "provider", "type": "enum", "title": "Provider", "default": "one", "options": ["one", "two", "three"]},
			{"key": "region", "type": "enum", "title": "Region", "default": "eu", "options": ["eu", "us"]}
		],
		"secrets": ` + secrets + `}`))
}

func TestASecretGroupIsReadFromTheManifest(t *testing.T) {
	m, err := readSecrets(`[{"name": "apiKey", "title": "API key", "group": "openrouter"}, {"name": "orgId", "title": "Organization"}]`)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Secrets[0].Group; got != "openrouter" {
		t.Errorf("group = %q, want openrouter", got)
	}
	if got := m.Secrets[1].Group; got != "" {
		t.Errorf("group = %q, want none", got)
	}
}

func TestADeclarationGroupIsTrimmedNonEmptyAndAtMost64Characters(t *testing.T) {
	cases := []struct {
		name    string
		group   string
		invalid bool
	}{
		{"a plain group", "openrouter", false},
		{"a group of 64 characters", strings.Repeat("g", 64), false},
		{"a group with an inner space", "my provider", false},
		{"a group of 65 characters", strings.Repeat("g", 65), true},
		{"a group with a leading space", " openrouter", true},
		{"a group with a trailing space", "openrouter ", true},
		{"a group of spaces", "   ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readSecrets(fmt.Sprintf(`[{"name": "apiKey", "title": "T", "group": %q}]`, tc.group))
			if (err != nil) != tc.invalid {
				t.Errorf("error = %v, want invalid: %v", err, tc.invalid)
			}
		})
	}
}

func TestTheSameNameIsValidOnlyForDisjointConditionsOnOneSetting(t *testing.T) {
	cases := []struct {
		name    string
		secrets string
		invalid bool
	}{
		{
			"disjoint lists of one setting",
			`[{"name": "apiKey", "title": "T", "group": "one", "when": {"setting": "provider", "is": ["one"]}},
			  {"name": "apiKey", "title": "T", "when": {"setting": "provider", "is": ["two", "three"]}}]`,
			false,
		},
		{
			"overlapping lists of one setting",
			`[{"name": "apiKey", "title": "T", "group": "one", "when": {"setting": "provider", "is": ["one", "two"]}},
			  {"name": "apiKey", "title": "T", "group": "two", "when": {"setting": "provider", "is": ["two", "three"]}}]`,
			true,
		},
		{
			"conditions on two settings",
			`[{"name": "apiKey", "title": "T", "group": "one", "when": {"setting": "provider", "is": ["one"]}},
			  {"name": "apiKey", "title": "T", "group": "two", "when": {"setting": "region", "is": ["us"]}}]`,
			true,
		},
		{
			"one declaration with no condition",
			`[{"name": "apiKey", "title": "T", "group": "one"},
			  {"name": "apiKey", "title": "T", "group": "two", "when": {"setting": "provider", "is": ["two"]}}]`,
			true,
		},
		{
			"the same name and group under disjoint lists",
			`[{"name": "apiKey", "title": "T", "group": "one", "when": {"setting": "provider", "is": ["one"]}},
			  {"name": "apiKey", "title": "T", "group": "one", "when": {"setting": "provider", "is": ["two"]}}]`,
			true,
		},
		{
			"the same name twice with no condition",
			`[{"name": "apiKey", "title": "T"}, {"name": "apiKey", "title": "T"}]`,
			true,
		},
		{
			"different names need no condition",
			`[{"name": "apiKey", "title": "T", "group": "one"}, {"name": "orgId", "title": "T", "group": "one"}]`,
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readSecrets(tc.secrets)
			if (err != nil) != tc.invalid {
				t.Errorf("error = %v, want invalid: %v", err, tc.invalid)
			}
		})
	}
}

func TestStoredSecretsIsNoLongerAReservedKey(t *testing.T) {
	if _, err := readSecrets(`[{"name": "storedSecrets", "title": "T"}]`); err != nil {
		t.Errorf("a secret named storedSecrets: %v", err)
	}
	_, err := manifestjson.Read(strings.NewReader(
		`{"id":"a.b","name":"A","version":"1.0.0","settings":[{"key":"storedSecrets","type":"string","title":"Stored"}]}`))
	if err != nil {
		t.Errorf("a setting named storedSecrets: %v", err)
	}
}
