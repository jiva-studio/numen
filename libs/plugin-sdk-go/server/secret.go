package server

// SecretAPIKey is the secret a plugin declares in its manifest for the key of
// the service it calls.
const SecretAPIKey = "apiKey"

// ReadSecret returns the key a Configure call hands over under the name of a
// secret the plugin declares, and whether the call carries that secret. An
// empty key means the host holds none, and the plugin drops the one it has.
func ReadSecret(values map[string]string, name string) (string, bool) {
	key, found := values[name]
	return key, found
}
