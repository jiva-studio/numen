package plugintest

import "testing"

func TestCheckConfigureAcceptsAKey(t *testing.T) {
	CheckConfigure(t, "theme", "sepia")
}

func TestCheckDependenciesAcceptsADeclaredModelWithAChecksum(t *testing.T) {
	manifest := []byte(`{"dependencies":[{"id":"model-a","kind":"model","sourceUrl":"https://example.invalid/a",` +
		`"checksumSha256":"` + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" + `"}]}`)
	CheckDependencies(t, manifest, true, "model-a")
}

func TestCheckDependenciesAcceptsAnEmptyIDList(t *testing.T) {
	CheckDependencies(t, []byte(`{}`), false)
}
