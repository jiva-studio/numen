// Package assembly defines window assembly interfaces.
package assembly

type runtimeAssembly interface {
	SettingsAssembly
	WorkerAssembly
	ScenarioAssembly
}

// Assembly is what this window requires from the application's composition root:
// the vault list and the installation's cache, the ports that access a vault, and
// the scenarios and passes that open one.
type Assembly interface {
	VaultAssembly
	IndexAssembly
	PluginAssembly
	runtimeAssembly
}
