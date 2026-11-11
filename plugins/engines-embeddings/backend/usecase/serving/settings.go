package serving

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/providers"
)

// Secret is the API key the host delivers. A call that does not carry the key leaves IsSet false and
// keeps the key the server holds; a call that carries it replaces the key, with none when Value is empty.
type Secret struct {
	Value string
	IsSet bool
}

// Update applies the settings values and the API key. The model the server serves follows them.
func (s *Server) Update(values map[string]string, apiKey Secret) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyProfileLocked(performance.Read(values))
	s.loadProfilesLocked()
	if provider := values["provider"]; provider != "" {
		s.provider = provider
	}
	model := values["model"]
	if model != "" && !s.profiles.IsLocal(model) {
		s.remoteModel = model
	}
	if providers.IsRemote(s.provider) && s.remoteModel == "" {
		s.remoteModel = providers.GetDefaultModel(s.provider)
	}
	s.modelInfo.MinChunkLength = chunking.ReadMinLength(values)
	if _, ok := values[chunking.TokensKey]; ok {
		s.chunkTokens = chunking.ReadTokens(values)
	}
	if apiKey.IsSet {
		s.apiKey = apiKey.Value
	}
	s.baseURL = values["baseUrl"]
	if !providers.IsRemote(s.provider) {
		s.updateLocalModelLocked(model)
	}
	s.refreshModelInfoLocked()
}

func (s *Server) loadProfilesLocked() {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="embedded shipped profiles json is valid"
	s.profiles, s.profilesErr = s.profileSrc.Load()
}
