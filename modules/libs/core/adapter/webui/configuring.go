package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Settings is every setting of the file a person configures this installation
// in, and the models the settings that name one can be set to.
func (a *API) Settings(
	_ context.Context, _ *connect.Request[v1.SettingsRequest],
) (*connect.Response[v1.SettingsResponse], error) {
	if a.Configuring.Configured == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	written, path, err := a.Configuring.Configured()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var models []port.Model
	if a.Configuring.Models != nil {
		models = a.Configuring.Models()
	}
	return connect.NewResponse(&v1.SettingsResponse{
		Written: written,
		Path:    path,
		Models:  offered(models),
	}), nil
}

// ChooseSettings writes settings into that file. A value the settings could not
// be read out of again is the client's to correct, and the file is left as it
// was.
func (a *API) ChooseSettings(
	_ context.Context, r *connect.Request[v1.ChooseSettingsRequest],
) (*connect.Response[v1.ChooseSettingsResponse], error) {
	if a.Configuring.ChoosesSetting == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	written := make([]port.Setting, 0, len(r.Msg.GetSettings()))
	for _, one := range r.Msg.GetSettings() {
		written = append(written, port.Setting{Path: one.GetAt(), JSON: one.GetValue()})
	}
	if err := a.Configuring.ChoosesSetting(written); err != nil {
		if errors.Is(err, port.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ChooseSettingsResponse{}), nil
}

// SettingsFile is that file as its person wrote it, byte for byte.
func (a *API) SettingsFile(
	_ context.Context, _ *connect.Request[v1.SettingsFileRequest],
) (*connect.Response[v1.SettingsFileResponse], error) {
	if a.Configuring.ConfiguredFile == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	written, path, err := a.Configuring.ConfiguredFile()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.SettingsFileResponse{Written: written, Path: path}), nil
}

// WriteSettingsFile replaces that file whole. A file the settings could not be
// read out of is the client's to correct, and the file is left as it was.
//
// The client presents the file it last read. A file holding bytes it has not
// read is left alone and refused `stale`: the person chooses what happens to
// their text.
func (a *API) WriteSettingsFile(
	_ context.Context, r *connect.Request[v1.WriteSettingsFileRequest],
) (*connect.Response[v1.WriteSettingsFileResponse], error) {
	if a.Configuring.WritesFile == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := a.Configuring.WritesFile(r.Msg.GetWritten(), r.Msg.Seen); err != nil {
		if errors.Is(err, port.ErrChanged) {
			stale := v1.Refusal_REFUSAL_STALE
			return connect.NewResponse(&v1.WriteSettingsFileResponse{Refusal: &stale}), nil
		}
		if errors.Is(err, port.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteSettingsFileResponse{}), nil
}

// offered is the models as the wire carries them.
func offered(held []port.Model) []*v1.Model {
	models := make([]*v1.Model, 0, len(held))
	for _, one := range held {
		models = append(models, &v1.Model{
			NamedAt:   one.Path,
			Name:      one.Name,
			Title:     one.Title,
			Shelf:     one.Shelf,
			ByDefault: one.Default,
			Writes:    writes(one.Writes),
			Presence:  standing[one.Presence],
		})
	}
	return models
}

// standing is what a model's files are on this machine, as the wire carries it.
var standing = map[port.Presence]v1.Presence{
	port.NothingToFetch: v1.Presence_PRESENCE_NOTHING_TO_FETCH,
	port.Present:        v1.Presence_PRESENCE_PRESENT,
	port.NotFetched:     v1.Presence_PRESENCE_NOT_FETCHED,
}

// writes is what choosing a model writes, as the wire carries it.
func writes(held []port.Setting) []*v1.Setting {
	written := make([]*v1.Setting, 0, len(held))
	for _, one := range held {
		written = append(written, &v1.Setting{At: one.Path, Value: one.JSON})
	}
	return written
}
