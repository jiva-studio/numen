package server

import "testing"

func TestReadSecret(t *testing.T) {
	cases := []struct {
		name      string
		values    map[string]string
		wantKey   string
		wantFound bool
	}{
		{"a stored key", map[string]string{SecretAPIKey: "sk-fixture", "model": "m"}, "sk-fixture", true},
		{"no key in the store", map[string]string{SecretAPIKey: ""}, "", true},
		{"a call without the secret", map[string]string{"model": "m"}, "", false},
		{"no values", nil, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, found := ReadSecret(tc.values, SecretAPIKey)
			if key != tc.wantKey || found != tc.wantFound {
				t.Errorf("ReadSecret = %q, %v, want %q, %v", key, found, tc.wantKey, tc.wantFound)
			}
		})
	}
}
