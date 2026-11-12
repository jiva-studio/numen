package service

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"

// Provider returns the roles the plugin fills for its connection type: complete and the lists of
// models and reasoning levels.
func (s *agentServer) Provider() connection.Provider {
	return connection.Provider{Completer: s.completion, Embedder: nil, Models: s.completion, Profiles: nil}
}
