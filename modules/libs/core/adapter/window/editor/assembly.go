package editor

// Assembly is what this window asks whoever put the application together: the
// vault list and the cache of the installation it serves, the ports it reaches
// a vault through, and the scenarios and passes it opens one with.
//
// Everything belonging to the installation is answered currently. What
// belongs to a vault is opened here and opened again each time another vault is
// shown.
//
// The cache is answered as the ports asked of it, so this window names no
// store. It is opened before the window is, and the window closes it.
type Assembly interface {
	VaultAssembly
	IndexAssembly
	PluginAssembly
	SettingsAssembly
	WorkerAssembly
	ScenarioAssembly
}
