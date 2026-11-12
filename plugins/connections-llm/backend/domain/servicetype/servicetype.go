// Package servicetype names the connection types of the plugin and the fields they hold.
package servicetype

// PluginID is the identity the manifest gives the plugin; a type id is the plugin id, a slash and the
// type.
const PluginID = "md.numen.connections.llm"

// The connection types of the plugin.
const (
	OpenRouter = "openrouter"
	OpenAI     = "openai"
	Compatible = "openai-compatible"
)

// The keys of the fields of a connection.
const (
	FieldBaseURL = "baseUrl"
	SecretAPIKey = "apiKey"
)

// IsKnown reports whether the plugin provides the type.
func IsKnown(typeID string) bool {
	return typeID == OpenRouter || typeID == OpenAI || typeID == Compatible
}
