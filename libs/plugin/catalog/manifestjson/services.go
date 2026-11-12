package manifestjson

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// listPluginServedServices returns, sorted, the services of the numen.v1 protocol that several plugins
// implement and a manifest may declare. Any other service of the protocol is served by the core itself.
func listPluginServedServices() []string {
	return []string{
		"numen.v1.AgentService",
		"numen.v1.EmbeddingService",
		"numen.v1.PluginToolService",
	}
}

func listServiceNameIssues(m manifest.Manifest) []error {
	var issues []error
	served := listPluginServedServices()
	for _, name := range m.Services {
		isOwn := name == m.ID || strings.HasPrefix(name, m.ID+".")
		if !isOwn && !slices.Contains(served, name) {
			issues = append(issues, fmt.Errorf("%w: %q starts with neither the plugin id %q nor the name of a service a plugin may serve (%s)",
				manifest.ErrInvalidService, name, m.ID, strings.Join(served, ", ")))
		}
	}
	return issues
}
