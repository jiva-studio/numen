// Package settings provides settings RPC handlers for the editor window.
package settings

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
)

// Handler answers every setting read and written, the settings file
// read and replaced whole, and the keys of the secrets plugins declare, stored
// and removed.
type Handler struct {
	Get          *setting.Get
	Write        *setting.Write
	GetFile      *setting.GetFile
	WriteFile    *setting.WriteFile
	WriteSecret  *setting.WriteSecret
	DeleteSecret *setting.DeleteSecret
	PluginRetry  *setting.RetryPlugin
}

func NewHandler() *Handler { return &Handler{} }

// errNoSettings is what a build that serves no settings answers.
var errNoSettings = errors.New("this build serves no settings")

// GetSettings is every setting of the file a person configures this
// installation in, and the models the settings that name one can be set to.
func (h *Handler) GetSettings(
	_ context.Context, _ *connect.Request[v1.GetSettingsRequest],
) (*connect.Response[v1.GetSettingsResponse], error) {
	if h.Get == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	got, err := h.Get.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetSettingsResponse{
		Written: got.Written,
		Path:    got.Path,
		Models:  NewWireModels(got.Models),
		NestedHeadingLimitBounds: &v1.Bounds{
			Least: got.NestedHeadingLimitBounds.Least,
			Most:  got.NestedHeadingLimitBounds.Most,
		},
		Plugins: NewWirePluginInfos(got.Plugins),
	}), nil
}

// WriteSettings writes settings into that file. A value the settings could not
// be read out of again is the client's to correct, and the file is left as it
// was.
func (h *Handler) WriteSettings(
	_ context.Context, r *connect.Request[v1.WriteSettingsRequest],
) (*connect.Response[v1.WriteSettingsResponse], error) {
	if h.Write == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	content := make([]setting.Setting, 0, len(r.Msg.GetSettings()))
	for _, one := range r.Msg.GetSettings() {
		content = append(content, setting.Setting{Path: one.GetAt(), JSON: one.GetValue()})
	}
	if err := h.Write.Execute(content); err != nil {
		if errors.Is(err, setting.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteSettingsResponse{}), nil
}

// ReadSettingsFile is that file as its person wrote it, byte for byte.
func (h *Handler) ReadSettingsFile(
	_ context.Context, _ *connect.Request[v1.ReadSettingsFileRequest],
) (*connect.Response[v1.ReadSettingsFileResponse], error) {
	if h.GetFile == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	got, err := h.GetFile.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.ReadSettingsFileResponse{Written: got.Written, Path: got.Path}), nil
}

// WriteSettingsFile replaces that file whole. A file the settings could not be
// read out of is the client's to correct, and the file is left as it was.
//
// The client presents the file it last read. A file holding bytes it has not
// their text.
func (h *Handler) WriteSettingsFile(
	_ context.Context, r *connect.Request[v1.WriteSettingsFileRequest],
) (*connect.Response[v1.WriteSettingsFileResponse], error) {
	if h.WriteFile == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := h.WriteFile.Execute(r.Msg.GetWritten(), r.Msg.Baseline); err != nil {
		if errors.Is(err, setting.ErrStale) {
			staleCode := v1.ErrorCode_ERROR_CODE_STALE
			return connect.NewResponse(&v1.WriteSettingsFileResponse{Error: &staleCode}), nil
		}
		if errors.Is(err, setting.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteSettingsFileResponse{}), nil
}

// NewWireModels is the models as the wire carries them.
func NewWireModels(target []setting.Model) []*v1.Model {
	models := make([]*v1.Model, 0, len(target))
	for _, one := range target {
		models = append(models, &v1.Model{
			NamedAt:          one.Path,
			Name:             one.Name,
			Title:            one.Title,
			Group:            one.Group,
			IsDefault:        one.IsDefault,
			Writes:           newWireSettings(one.Writes),
			Presence:         Presences[one.Presence],
			ReasoningLevels:  newWireReasoningLevels(one.ReasoningLevels),
			DefaultReasoning: one.DefaultReasoning,
		})
	}
	return models
}

// Presences is what a model's files are on this machine, as the wire carries
// it.
var Presences = map[setting.Presence]v1.Presence{
	setting.NothingToFetch: v1.Presence_PRESENCE_NOTHING_TO_FETCH,
	setting.Present:        v1.Presence_PRESENCE_PRESENT,
	setting.NotFetched:     v1.Presence_PRESENCE_NOT_FETCHED,
}

// newWireSettings is what choosing a model writes, as the wire carries it.
func newWireSettings(target []setting.Setting) []*v1.Setting {
	content := make([]*v1.Setting, 0, len(target))
	for _, one := range target {
		content = append(content, &v1.Setting{At: one.Path, Value: one.JSON})
	}
	return content
}

// NewWirePluginInfos is the discovered plugins and their declarative settings, as the wire carries it.
func NewWirePluginInfos(target []setting.PluginInfo) []*v1.PluginInfo {
	infos := make([]*v1.PluginInfo, 0, len(target))
	for _, one := range target {
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
				ModelsOf:    s.ModelsOf,
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
			CompletionConsent: Consents[one.Consent],
			Failure:           newWirePluginFailure(one.Failure),
			HasFrontend:       one.HasFrontend,
		})
	}
	return infos
}

// Consents is whether the person lets a plugin ask a model, as the wire carries it.
var Consents = map[setting.Consent]v1.CompletionConsent{
	setting.ConsentNone:       v1.CompletionConsent_COMPLETION_CONSENT_UNSPECIFIED,
	setting.ConsentUnanswered: v1.CompletionConsent_COMPLETION_CONSENT_UNANSWERED,
	setting.ConsentAsked:      v1.CompletionConsent_COMPLETION_CONSENT_ASKED,
	setting.ConsentAllowed:    v1.CompletionConsent_COMPLETION_CONSENT_ALLOWED,
	setting.ConsentRefused:    v1.CompletionConsent_COMPLETION_CONSENT_REFUSED,
}

// WritePluginSecret stores the key of a secret a plugin declares. A secret the
// plugin does not declare, and a machine without a key store, are the client's
// to be told of.
func (h *Handler) WritePluginSecret(
	ctx context.Context, r *connect.Request[v1.WritePluginSecretRequest],
) (*connect.Response[v1.WritePluginSecretResponse], error) {
	if h.WriteSecret == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := h.WriteSecret.Execute(ctx, r.Msg.GetPluginId(), r.Msg.GetName(), r.Msg.GetKey()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.WritePluginSecretResponse{}), nil
}

// DeletePluginSecret removes the key, if any, of a secret a plugin declares.
func (h *Handler) DeletePluginSecret(
	ctx context.Context, r *connect.Request[v1.DeletePluginSecretRequest],
) (*connect.Response[v1.DeletePluginSecretResponse], error) {
	if h.DeleteSecret == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := h.DeleteSecret.Execute(ctx, r.Msg.GetPluginId(), r.Msg.GetName()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&v1.DeletePluginSecretResponse{}), nil
}

// newWirePluginFailure is why the host suspends a plugin, as the wire carries it, and nil for a plugin it does not.
func newWirePluginFailure(failure *setting.PluginFailure) *v1.PluginFailure {
	if failure == nil {
		return nil
	}
	return &v1.PluginFailure{Reason: failure.Reason, Detail: failure.Detail, FailedAtUnixMs: failure.At.UnixMilli()}
}

// RetryPlugin starts a plugin the host suspends once more. A plugin that fails
// again is answered with its reason, and one the host does not know is not found.
func (h *Handler) RetryPlugin(
	ctx context.Context, r *connect.Request[v1.RetryPluginRequest],
) (*connect.Response[v1.RetryPluginResponse], error) {
	if h.PluginRetry == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	failure, err := h.PluginRetry.Execute(ctx, r.Msg.GetPluginId())
	if err != nil {
		var unavailable *setting.UnavailableError
		if errors.As(err, &unavailable) && unavailable.Reason == setting.PluginNotInstalled {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	reply := &v1.RetryPluginResponse{}
	if failure != nil {
		reply.Failure = failure.Reason
	}
	return connect.NewResponse(reply), nil
}

// newWireReasoningLevels is the levels of reasoning as the wire carries them.
func newWireReasoningLevels(target []setting.ReasoningLevel) []*v1.ReasoningLevel {
	if len(target) == 0 {
		return nil
	}
	levels := make([]*v1.ReasoningLevel, 0, len(target))
	for _, one := range target {
		levels = append(levels, &v1.ReasoningLevel{Name: one.Name, Title: one.Title})
	}
	return levels
}
