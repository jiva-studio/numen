package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

// settingsHandler answers every setting read and written, the settings file
// read and replaced whole, and the keys of the secrets plugins declare, stored
// and removed.
type settingsHandler struct {
	get          *settings.Get
	write        *settings.Write
	getFile      *settings.GetFile
	writeFile    *settings.WriteFile
	writeSecret  *settings.WriteSecret
	deleteSecret *settings.DeleteSecret
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
		Plugins:            newWirePluginInfos(got.Plugins),
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

// newWirePluginInfos is the discovered plugins and their declarative settings, as the wire carries it.
func newWirePluginInfos(held []settings.PluginInfo) []*v1.PluginInfo {
	infos := make([]*v1.PluginInfo, 0, len(held))
	for _, one := range held {
		pluginSettings := make([]*v1.PluginSetting, 0, len(one.Settings))
		for _, s := range one.Settings {
			pluginSettings = append(pluginSettings, &v1.PluginSetting{
				Key:         s.Key,
				Type:        s.Type,
				Title:       s.Title,
				Description: s.Description,
				DefaultJson: s.DefaultJSON,
				Palette:     s.Palette,
				Options:     s.Options,
				Min:         s.Min,
				Max:         s.Max,
				Step:        s.Step,
			})
		}
		infos = append(infos, &v1.PluginInfo{
			Id:           one.ID,
			Name:         one.Name,
			Version:      one.Version,
			Description:  one.Description,
			Capabilities: one.Capabilities,
			Settings:     pluginSettings,
			IsActive:     one.IsActive,
			Permissions:  one.Permissions,
			Secrets:      one.Secrets,
			// The names of the stored secrets travel, and a key never does.
			StoredSecrets:     one.StoredSecrets,
			Models:            newWirePluginModels(one.Models),
			CompletionConsent: consents[one.Consent],
		})
	}
	return infos
}

// newWirePluginModels is the models a plugin answers completions with, as the wire carries them.
func newWirePluginModels(held []settings.AgentModel) []*v1.PluginModel {
	models := make([]*v1.PluginModel, 0, len(held))
	for _, one := range held {
		models = append(models, &v1.PluginModel{Name: one.Name, Title: one.Title, IsDefault: one.IsDefault})
	}
	return models
}

// consents is whether the person lets a plugin ask a model, as the wire carries it.
var consents = map[settings.Consent]v1.CompletionConsent{
	settings.ConsentNone:       v1.CompletionConsent_COMPLETION_CONSENT_UNSPECIFIED,
	settings.ConsentUnanswered: v1.CompletionConsent_COMPLETION_CONSENT_UNANSWERED,
	settings.ConsentAsked:      v1.CompletionConsent_COMPLETION_CONSENT_ASKED,
	settings.ConsentAllowed:    v1.CompletionConsent_COMPLETION_CONSENT_ALLOWED,
	settings.ConsentRefused:    v1.CompletionConsent_COMPLETION_CONSENT_REFUSED,
}

// WritePluginSecret stores the key of a secret a plugin declares. A secret the
// plugin does not declare, and a machine without a key store, are the client's
// to be told of.
func (h *settingsHandler) WritePluginSecret(
	ctx context.Context, r *connect.Request[v1.WritePluginSecretRequest],
) (*connect.Response[v1.WritePluginSecretResponse], error) {
	if h.writeSecret == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := h.writeSecret.Execute(ctx, r.Msg.GetPluginId(), r.Msg.GetName(), r.Msg.GetKey()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.WritePluginSecretResponse{}), nil
}

// DeletePluginSecret removes the key, if any, of a secret a plugin declares.
func (h *settingsHandler) DeletePluginSecret(
	ctx context.Context, r *connect.Request[v1.DeletePluginSecretRequest],
) (*connect.Response[v1.DeletePluginSecretResponse], error) {
	if h.deleteSecret == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := h.deleteSecret.Execute(ctx, r.Msg.GetPluginId(), r.Msg.GetName()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.DeletePluginSecretResponse{}), nil
}
