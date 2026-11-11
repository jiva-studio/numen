package plugin

// Binding gives the host the mode and the vault path the plugins run under.
type Binding struct {
	teardown *Teardown
	mode     *Mode
}

func NewBinding(teardown *Teardown, mode *Mode) Binding {
	return Binding{teardown: teardown, mode: mode}
}

// SetSafeMode enables or disables safe recovery mode. Entering it stops every running plugin.
func (b Binding) SetSafeMode(enabled bool) { b.teardown.SetSafeMode(enabled) }

// SetVaultPath sets the active vault root path for path traversal checks.
func (b Binding) SetVaultPath(path string) { b.mode.SetVaultPath(path) }
