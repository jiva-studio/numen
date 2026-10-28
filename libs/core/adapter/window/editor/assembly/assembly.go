// Package assembly defines window assembly interfaces.
package assembly

type runtimeAssembly interface {
	SettingsAssembly
	WorkerAssembly
	ScenarioAssembly
}

// Assembly is what this window asks whoever put the application together: the
// vault list and the cache of the installation it serves, the ports it reaches
// a vault through, and the scenarios and passes it opens one with.
type Assembly interface {
	VaultAssembly
	IndexAssembly
	PluginAssembly
	runtimeAssembly
}
