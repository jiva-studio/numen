// Package endpoint holds the HTTP chat service a configured provider points at.
package endpoint

// DefaultModel is the model asked over HTTP when the settings name none.
const DefaultModel = "openai/gpt-4o-mini"

// Endpoint is the HTTP chat service a configured provider points at.
type Endpoint struct {
	Provider string
	BaseURL  string
	Model    string
	APIKey   string
}

// GetDefault has the host answer every question.
func GetDefault() Endpoint {
	return Endpoint{Provider: "agent", BaseURL: "", Model: DefaultModel, APIKey: ""}
}

// WithSettings returns the endpoint with the provider, base URL and model the settings name. A
// provider or a model the settings leave empty stays as it is.
func (e Endpoint) WithSettings(values map[string]string) Endpoint {
	if provider := values["provider"]; provider != "" {
		e.Provider = provider
	}
	if baseURL, ok := values["baseUrl"]; ok {
		e.BaseURL = baseURL
	}
	if name := values["model"]; name != "" {
		e.Model = name
	}
	return e
}
