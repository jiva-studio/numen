// Package providers names the services that give the plugin its vectors.
package providers

// The provider that runs the model on this machine, and the two hosted services with an address of
// their own.
const (
	Local      = "local"
	OpenAI     = "openai"
	OpenRouter = "openrouter"
)

const (
	modelOpenAI     = "text-embedding-3-small"
	modelOpenRouter = "openai/text-embedding-3-small"
)

// IsRemote returns whether provider names a service other than this machine. A provider that is not
// set is not remote.
func IsRemote(provider string) bool {
	return provider != "" && provider != Local
}

// GetDefaultModel is the model a remote provider serves when the settings name none.
func GetDefaultModel(provider string) string {
	if provider == OpenRouter {
		return modelOpenRouter
	}
	return modelOpenAI
}

// GetServiceURL is the address of a hosted provider, and empty for a provider whose address the
// settings name.
func GetServiceURL(provider string) string {
	switch provider {
	case OpenAI:
		return "https://api.openai.com/v1"
	case OpenRouter:
		return "https://openrouter.ai/api/v1"
	}
	return ""
}
