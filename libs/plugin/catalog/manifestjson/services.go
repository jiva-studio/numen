package manifestjson

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// pluginServedProtocolServices are the services of the numen.v1 protocol that several plugins implement and a
// manifest may declare. Any other service of the protocol is served by the core itself.
var pluginServedProtocolServices = map[string]bool{
	"numen.v1.AgentService":      true,
	"numen.v1.EmbeddingService":  true,
	"numen.v1.PluginToolService": true,
}

func listServiceNameIssues(m manifest.Manifest) []error {
	var issues []error
	for _, name := range m.Services {
		isOwn := name == m.ID || strings.HasPrefix(name, m.ID+".")
		if !isOwn && !pluginServedProtocolServices[name] {
			issues = append(issues, fmt.Errorf("%w: %q starts with neither the plugin id %q nor the name of a service a plugin may serve (%s)",
				manifest.ErrInvalidService, name, m.ID, listServedServiceNames()))
		}
	}
	return issues
}

func listServedServiceNames() string {
	names := make([]string, 0, len(pluginServedProtocolServices))
	for name := range pluginServedProtocolServices {
		names = append(names, name)
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}
