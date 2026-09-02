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
	if a.Configured == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	written, path, err := a.Configured()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var models []port.Model
	if a.Models != nil {
		models = a.Models()
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
	if a.ChoosesSetting == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	written := make([]port.Setting, 0, len(r.Msg.GetSettings()))
	for _, one := range r.Msg.GetSettings() {
		written = append(written, port.Setting{At: one.GetAt(), Value: one.GetValue()})
	}
	if err := a.ChoosesSetting(written); err != nil {
		if errors.Is(err, port.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ChooseSettingsResponse{}), nil
}

// offered is the models as the wire carries them.
func offered(held []port.Model) []*v1.Model {
	models := make([]*v1.Model, 0, len(held))
	for _, one := range held {
		models = append(models, &v1.Model{
			NamedAt:   one.NamedAt,
			Name:      one.Name,
			Title:     one.Title,
			Shelf:     one.Shelf,
			ByDefault: one.ByDefault,
			Writes:    writes(one.Writes),
		})
	}
	return models
}

// writes is what choosing a model writes, as the wire carries it.
func writes(held []port.Setting) []*v1.Setting {
	written := make([]*v1.Setting, 0, len(held))
	for _, one := range held {
		written = append(written, &v1.Setting{At: one.At, Value: one.Value})
	}
	return written
}
