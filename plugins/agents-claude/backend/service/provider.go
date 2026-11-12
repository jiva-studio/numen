package service

import "github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"

// Provider returns the roles the plugin fills for its connection type: complete and the lists of
// models and reasoning levels.
func (a *agent) Provider() connection.Provider {
	return connection.Provider{Completer: a.completion, Embedder: nil, Models: a.completion, Profiles: nil}
}
