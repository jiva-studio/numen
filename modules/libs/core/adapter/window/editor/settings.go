package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

// settingsHandler answers every setting read and written, and the settings file
// read and replaced whole.
type settingsHandler struct {
	get       *settings.Get
	write     *settings.Write
	getFile   *settings.GetFile
	writeFile *settings.WriteFile
}

// errNoSettings is what a build that serves no settings answers.
var errNoSettings = errors.New("this build serves no settings")

// GetSettings is every setting of the file a person configures this
// installation in, and the models the settings that name one can be set to.
func (h *settingsHandler) GetSettings(
	_ context.Context, _ *connect.Request[v1.GetSettingsRequest],
) (*connect.Response[v1.GetSettingsResponse], error) {
	if h.get == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	got, err := h.get.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetSettingsResponse{
		Written: got.Written,
		Path:    got.Path,
		Models:  newWireModels(got.Models),
		NestedHeadingLimitBounds: &v1.Bounds{
			Least: got.NestedHeadingLimitBounds.Least,
			Most:  got.NestedHeadingLimitBounds.Most,
		},
		LatestDayStartTime: got.LatestDayStartTime,
		Day:                got.Day,
	}), nil
}

// WriteSettings writes settings into that file. A value the settings could not
// be read out of again is the client's to correct, and the file is left as it
// was.
func (h *settingsHandler) WriteSettings(
	_ context.Context, r *connect.Request[v1.WriteSettingsRequest],
) (*connect.Response[v1.WriteSettingsResponse], error) {
	if h.write == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	written := make([]settings.Setting, 0, len(r.Msg.GetSettings()))
	for _, one := range r.Msg.GetSettings() {
		written = append(written, settings.Setting{Path: one.GetAt(), JSON: one.GetValue()})
	}
	if err := h.write.Execute(written); err != nil {
		if errors.Is(err, settings.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteSettingsResponse{}), nil
}

// ReadSettingsFile is that file as its person wrote it, byte for byte.
func (h *settingsHandler) ReadSettingsFile(
	_ context.Context, _ *connect.Request[v1.ReadSettingsFileRequest],
) (*connect.Response[v1.ReadSettingsFileResponse], error) {
	if h.getFile == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	got, err := h.getFile.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ReadSettingsFileResponse{Written: got.Written, Path: got.Path}), nil
}

// WriteSettingsFile replaces that file whole. A file the settings could not be
// read out of is the client's to correct, and the file is left as it was.
//
// The client presents the file it last read. A file holding bytes it has not
// read is left alone and refused `stale`: the person chooses what happens to
// their text.
func (h *settingsHandler) WriteSettingsFile(
	_ context.Context, r *connect.Request[v1.WriteSettingsFileRequest],
) (*connect.Response[v1.WriteSettingsFileResponse], error) {
	if h.writeFile == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := h.writeFile.Execute(r.Msg.GetWritten(), r.Msg.Baseline); err != nil {
		if errors.Is(err, settings.ErrStale) {
			stale := v1.ErrorCode_ERROR_CODE_STALE
			return connect.NewResponse(&v1.WriteSettingsFileResponse{Error: &stale}), nil
		}
		if errors.Is(err, settings.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteSettingsFileResponse{}), nil
}

// newWireModels is the models as the wire carries them.
func newWireModels(held []settings.Model) []*v1.Model {
	models := make([]*v1.Model, 0, len(held))
	for _, one := range held {
		models = append(models, &v1.Model{
			NamedAt:   one.Path,
			Name:      one.Name,
			Title:     one.Title,
			Group:     one.Group,
			IsDefault: one.IsDefault,
			Writes:    newWireSettings(one.Writes),
			Presence:  presences[one.Presence],
		})
	}
	return models
}

// presences is what a model's files are on this machine, as the wire carries
// it.
var presences = map[settings.Presence]v1.Presence{
	settings.NothingToFetch: v1.Presence_PRESENCE_NOTHING_TO_FETCH,
	settings.Present:        v1.Presence_PRESENCE_PRESENT,
	settings.NotFetched:     v1.Presence_PRESENCE_NOT_FETCHED,
}

// newWireSettings is what choosing a model writes, as the wire carries it.
func newWireSettings(held []settings.Setting) []*v1.Setting {
	written := make([]*v1.Setting, 0, len(held))
	for _, one := range held {
		written = append(written, &v1.Setting{At: one.Path, Value: one.JSON})
	}
	return written
}
