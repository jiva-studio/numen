package editor

import (
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/command"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/plugincatalog"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/search"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/texts"
)

// NewNoteRoute is the route of the notes service.
func NewNoteRoute(state *State, handler *NoteHandler) Route {
	path, served := numenv1connect.NewNoteServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewFileRoute is the route of the files service.
func NewFileRoute(state *State, handler *FileHandler) Route {
	path, served := numenv1connect.NewFileServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewVaultsRoute is the route of the service that lists and changes the vaults.
func NewVaultsRoute(state *State, handler *VaultsHandler) Route {
	path, served := numenv1connect.NewVaultsServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewMediaRoute is the route of the service that tells where a media file is played from.
func NewMediaRoute(state *State, handler *MediaHandler) Route {
	path, served := numenv1connect.NewMediaServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewSearchRoute is the route of the search service.
func NewSearchRoute(state *State, handler *search.Handler) Route {
	path, served := numenv1connect.NewSearchServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewSettingsRoute is the route of the settings service.
func NewSettingsRoute(state *State, handler *settings.Handler) Route {
	path, served := numenv1connect.NewSettingsServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewCommandRoute is the route of the service that lists and runs the commands of plugins.
func NewCommandRoute(state *State, handler *command.Handler) Route {
	path, served := numenv1connect.NewCommandServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewTextRoute is the route of the service that reads the text of files.
func NewTextRoute(state *State, handler *texts.Handler) Route {
	path, served := numenv1connect.NewTextServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewCatalogRoute is the route of the service that lists the plugins the application has.
func NewCatalogRoute(state *State, handler *plugincatalog.Handler) Route {
	path, served := numenv1connect.NewCatalogServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}

// NewPluginRoute is the route of the service plugins are managed through, which this window does
// not implement.
func NewPluginRoute(state *State) Route {
	handler := numenv1connect.UnimplementedPluginServiceHandler{}
	path, served := numenv1connect.NewPluginServiceHandler(handler, state.CountQuestions())
	return NewRoute(path, served, handler)
}
