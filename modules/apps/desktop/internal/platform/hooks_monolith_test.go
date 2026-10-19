//go:build plugins_monolith

package platform

import (
	"net/http"
	"testing"
)

const exampleManifest = `{
	"id": "md.example.reader",
	"name": "Example Reader",
	"version": "1.0.0",
	"capabilities": ["document"],
	"targets": ["desktop.editor"],
	"permissions": [],
	"backend": {"type": "process", "entrypoint": "bin/example", "serviceName": "numen.v1.DocumentService"},
	"settings": []
}`

// A plugin registered before the configuration is made is a plugin of its host.
func TestARegisteredPluginIsAPluginOfTheHost(t *testing.T) {
	saved, wasMonolith := builtInPlugins, IsMonolith()
	t.Cleanup(func() {
		builtInPlugins = saved
		SetMonolith(wasMonolith)
	})
	RegisterPlugin([]byte(exampleManifest), func(map[string]string) (http.Handler, error) {
		return http.NewServeMux(), nil
	})

	if _, ok := GetConfig().PluginHost.GetPlugin("md.example.reader"); !ok {
		t.Fatal("the registered plugin is not a plugin of the host")
	}
}
