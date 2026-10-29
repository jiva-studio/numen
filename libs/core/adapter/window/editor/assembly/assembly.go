// Package assembly defines window assembly interfaces.
package assembly

type runtimeAssembly interface {
	SettingsAssembly
	WorkerAssembly
	ScenarioAssembly
}

// Assembly is what this window requires from the application's composition root:
// the vault list and the installation's cache, the ports used to access a vault, and
// the scenarios and passes used to open one.
type Assembly interface {
	VaultAssembly
	IndexAssembly
	PluginAssembly
	runtimeAssembly
}
