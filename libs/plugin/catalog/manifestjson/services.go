package manifestjson

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func listPluginServedServices() []string {
	// Several plugins implement these; any other service of the numen.v1 protocol is served by the core itself.
	return []string{
		"numen.v1.AgentService",
		"numen.v1.EmbeddingService",
		"numen.v1.PluginToolService",
		"numen.v1.ProviderService",
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
