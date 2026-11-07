package server

// SecretAPIKey is the secret a plugin declares in its manifest for the key of
// the service it calls.
const SecretAPIKey = "apiKey"

// Secret is one allowed secret with its value.
type Secret struct{ ID, Name, Group, Value string }

// Secrets holds, for each active declaration of the plugin, the allowed secrets in the order the
// host chose.
type Secrets map[string][]Secret

// ReadSecrets returns the secrets a Configure call passes for a declaration, and whether the call
// carries that declaration. An empty list means no secret, and the plugin discards the ones it has.
func ReadSecrets(secrets Secrets, name string) ([]Secret, bool) {
	list, found := secrets[name]
	if !found {
		return nil, false
	}
	return list, true
}
