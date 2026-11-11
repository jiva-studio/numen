package windowsession

import "io"

// Scenarios are the use cases of one window, built over its Runtime.
type Scenarios struct {
	Switching Switching
	Closing   Closing
	Start     StartWindow
	ShowTab   ShowTab
	Import    ImportFiles
}

// Switching changes the vault the window shows.
type Switching struct {
	Show       ShowVault
	OpenPlugin OpenPluginVault
	Rebuild    RebuildSession
}

// Closing ends the window.
type Closing struct {
	Flush FlushPending
	Close CloseWindow
}

// NewScenarios makes the use cases of the window over rt and deps.
func NewScenarios(rt *Runtime, deps Deps) *Scenarios {
	if deps.Out == nil {
		deps.Out = io.Discard
	}
	life := &lifecycle{rt: rt, deps: deps}
	rebuild := RebuildSession{life: life}
	return &Scenarios{
		Switching: Switching{Show: ShowVault{life: life}, OpenPlugin: OpenPluginVault{life: life}, Rebuild: rebuild},
		Closing:   Closing{Flush: FlushPending{life: life}, Close: CloseWindow{life: life}},
		Start:     StartWindow{life: life, rebuild: rebuild},
		ShowTab:   ShowTab{life: life},
		Import:    NewImportFiles(deps.Vault.Import, deps.Window.Writes, deps.Window.Tasks, deps.Window.Changes),
	}
}
