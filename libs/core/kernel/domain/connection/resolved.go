package connection

import (
	"maps"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Resolved is a connection as one call to its provider reads it: the fields as text, with the fixed
// values of the type, and no value of a secret. ParallelRequests is the number of calls the connection
// runs at once: the stored field, else the default of the type, else 0 for no limit. Fingerprint
// identifies the type and the address the connection had when it was read.
type Resolved struct {
	ID               string
	Name             string
	Revision         int
	PluginID         string
	Declaration      manifest.ConnectionType
	Fields           map[string]string
	ParallelRequests int
	Fingerprint      string
}

// NewCall makes the call to the provider of the connection with the secrets read for it, the model and
// the reasoning level of the row.
func (r Resolved) NewCall(secrets map[string]string, model, reasoning string) modelcall.ProviderCall {
	return modelcall.ProviderCall{
		Type:      r.Declaration.ID,
		Fields:    maps.Clone(r.Fields),
		Secrets:   maps.Clone(secrets),
		Model:     model,
		Reasoning: reasoning,
	}
}
