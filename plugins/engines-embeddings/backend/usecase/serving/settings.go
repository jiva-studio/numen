package serving

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/chunking"
)

// The values of the setting source.
const (
	SourceLocal      = "local"
	SourceConnection = "connection"
)

// Update applies the settings values. The model the server serves follows them.
func (s *Server) Update(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyProfileLocked(performance.Read(values))
	s.loadProfilesLocked()
	s.source = readSource(values)
	s.connection.reset()
	s.modelInfo.MinChunkLength = chunking.ReadMinLength(values)
	if _, ok := values[chunking.TokensKey]; ok {
		s.chunkTokens = chunking.ReadTokens(values)
	}
	if !s.isConnectionLocked() {
		s.updateLocalModelLocked(values["model"])
	}
	s.refreshModelInfoLocked()
}

func (s *Server) loadProfilesLocked() {
	// gomutants:disable-next-line STATEMENT_REMOVE reason="embedded shipped profiles json is valid"
	s.profiles, s.profilesErr = s.profileSrc.Load(func(file string, err error) {
		s.report("embeddings: skipped the model profile "+file, err)
	})
}

func readSource(values map[string]string) string {
	if values["source"] == SourceConnection {
		return SourceConnection
	}
	return SourceLocal
}

func (s *Server) isConnectionLocked() bool {
	return s.source == SourceConnection
}
