package webui

import (
	"context"
	"encoding/json"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/refusal"
)

// errNotJSON is a value the file cannot hold.
var errNotJSON = errors.New("a setting is written as JSON")

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

// ChooseSetting writes settings into that file.
func (a *API) ChooseSetting(
	_ context.Context, r *connect.Request[v1.ChooseSettingRequest],
) (*connect.Response[v1.ChooseSettingResponse], error) {
	if a.ChoosesSetting == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	written := make([]port.Setting, 0, len(r.Msg.GetSettings()))
	for _, one := range r.Msg.GetSettings() {
		// A value that is not JSON is the client's to correct, and the file is
		// left as it was.
		if !json.Valid([]byte(one.GetValue())) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errNotJSON)
		}
		written = append(written, port.Setting{At: one.GetAt(), Value: one.GetValue()})
	}
	if err := a.ChoosesSetting(written); err != nil {
		reason, refused := refusal.By(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ChooseSettingResponse{Refusal: &reason}), nil
	}
	return connect.NewResponse(&v1.ChooseSettingResponse{}), nil
}

// offered is the models as the wire carries them.
func offered(held []port.Model) []*v1.NamedModel {
	models := make([]*v1.NamedModel, 0, len(held))
	for _, one := range held {
		models = append(models, &v1.NamedModel{
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
func writes(held []port.Setting) []*v1.Written {
	written := make([]*v1.Written, 0, len(held))
	for _, one := range held {
		written = append(written, &v1.Written{At: one.At, Value: one.Value})
	}
	return written
}
