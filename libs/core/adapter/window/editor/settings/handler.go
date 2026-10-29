// Package settings provides settings RPC handlers for the editor window.
package settings

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	setting "github.com/jiva-studio/numen/libs/core/usecase/settings"
)

// Handler answers every setting read and written, and the keys of the secrets
// plugins declare, stored and removed.
type Handler struct {
	Get          *setting.Get
	Write        *setting.Write
	WriteSecret  *setting.WriteSecret
	DeleteSecret *setting.DeleteSecret
	PluginRetry  *setting.RetryPlugin
	// PluginData and Vaults are nil where the installation keeps no text versions.
	PluginData PluginData
	Vaults     VaultProvider
	// ChoosePath is nil where the application has no window to put a picker in.
	ChoosePath *setting.ChoosePluginPath
	// OnPluginChange is called when a plugin setting or secret is updated.
	OnPluginChange func()
}

func NewHandler() *Handler { return &Handler{} }

// errNoSettings is what a build that serves no settings answers.
var errNoSettings = errors.New("this build serves no settings")

// GetSettings returns every setting of the file this installation is configured in,
// and the models that settings naming a model can be set to.
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
	hasPluginEdit := false
	for _, one := range r.Msg.GetSettings() {
		if len(one.GetAt()) >= 2 && one.GetAt()[0] == "plugins" {
			hasPluginEdit = true
		}
		content = append(content, setting.Setting{Path: one.GetAt(), JSON: one.GetValue()})
	}
	if err := h.Write.Execute(content); err != nil {
		if errors.Is(err, setting.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if hasPluginEdit && h.OnPluginChange != nil {
		h.OnPluginChange()
	}
	return connect.NewResponse(&v1.WriteSettingsResponse{}), nil
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
				Icon:        s.Icon,
				Options:     newWireOptions(s.Options),
				Min:         s.Min,
				Max:         s.Max,
				Step:        s.Step,
				ModelsOf:    s.ModelsOf,
				OptionsBy:   newWireOptionsBy(s.OptionsBy),
				When:        newWireCondition(s.When),
				Accept:      s.Accept,
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
			Secrets:      newWireSecrets(one.Secrets),
			// The names of the stored secrets travel, and a key never does.
			StoredSecrets:     one.StoredSecrets,
			CompletionConsent: Consents[one.Consent],
			Failure:           newWirePluginFailure(one.Failure),
			HasFrontend:       one.HasFrontend,
			Readers:           newWireReaders(one.Readers),
			SettingsViews:     newWireSettingsViews(one.SettingsViews),
		})
	}
	return infos
}

// Consents maps whether the user lets a plugin query a model to its wire value.
var Consents = map[setting.Consent]v1.CompletionConsent{
	setting.ConsentNone:      v1.CompletionConsent_COMPLETION_CONSENT_UNSPECIFIED,
	setting.ConsentPending:   v1.CompletionConsent_COMPLETION_CONSENT_PENDING,
	setting.ConsentRequested: v1.CompletionConsent_COMPLETION_CONSENT_REQUESTED,
	setting.ConsentAllowed:   v1.CompletionConsent_COMPLETION_CONSENT_ALLOWED,
	setting.ConsentDenied:    v1.CompletionConsent_COMPLETION_CONSENT_DENIED,
}

// WritePluginSecret stores the key of a secret a plugin declares. A secret the
// plugin does not declare, and a machine without a key store, are reported to
// the client.
func (h *Handler) WritePluginSecret(
	ctx context.Context, r *connect.Request[v1.WritePluginSecretRequest],
) (*connect.Response[v1.WritePluginSecretResponse], error) {
	if h.WriteSecret == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := h.WriteSecret.Execute(ctx, r.Msg.GetPluginId(), r.Msg.GetName(), r.Msg.GetKey()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if h.OnPluginChange != nil {
		h.OnPluginChange()
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
	if h.OnPluginChange != nil {
		h.OnPluginChange()
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

// newWireOptions is the options of a setting as the wire carries them.
func newWireOptions(target setting.PluginSettingOptions) []*v1.PluginSettingOption {
	if len(target) == 0 {
		return nil
	}
	options := make([]*v1.PluginSettingOption, 0, len(target))
	for _, one := range target {
		options = append(options, newWireOption(one))
	}
	return options
}

func newWireOption(one setting.PluginSettingOption) *v1.PluginSettingOption {
	return &v1.PluginSettingOption{Value: one.Value, Title: one.Title, Detail: one.Detail}
}

// newWireOptionsBy is the options a setting takes from another, as the wire carries them.
func newWireOptionsBy(target *setting.PluginSettingOptionsBy) *v1.PluginSettingOptionsBy {
	if target == nil {
		return nil
	}
	options := make(map[string]*v1.PluginSettingOptions, len(target.Options))
	for value, list := range target.Options {
		options[value] = &v1.PluginSettingOptions{Options: newWireOptions(list)}
	}
	return &v1.PluginSettingOptionsBy{Setting: target.Setting, Options: options}
}

// newWireCondition is the condition of a setting as the wire carries it.
func newWireCondition(target *setting.PluginSettingCondition) *v1.PluginSettingCondition {
	if target == nil {
		return nil
	}
	return &v1.PluginSettingCondition{Setting: target.Setting, Is: target.Is}
}

// newWireSecrets is the secrets a plugin declares, as the wire carries them.
func newWireSecrets(target []setting.PluginSecret) []*v1.PluginSecret {
	secrets := make([]*v1.PluginSecret, 0, len(target))
	for _, one := range target {
		secrets = append(secrets, &v1.PluginSecret{
			Name: one.Name, Title: one.Title, Description: one.Description, When: newWireCondition(one.When),
		})
	}
	return secrets
}

// ChoosePluginPath shows the machine's native picker for a plugin's path setting
// and writes the chosen path. A path the setting does not accept returns the reason.
func (h *Handler) ChoosePluginPath(
	ctx context.Context, r *connect.Request[v1.ChoosePluginPathRequest],
) (*connect.Response[v1.ChoosePluginPathResponse], error) {
	if h.ChoosePath == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	chosen, err := h.ChoosePath.Execute(ctx, r.Msg.GetPluginId(), r.Msg.GetKey())
	reply := &v1.ChoosePluginPathResponse{Path: chosen.Path, IsChosen: chosen.IsChosen}
	switch {
	case errors.Is(err, setting.ErrNotASetting):
		reply.Problem = strings.TrimPrefix(err.Error(), setting.ErrNotASetting.Error()+": ")
	case errors.Is(err, setting.ErrNoPathSetting):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(reply), nil
}

// newWireSettingsViews is the settings views of a plugin as the wire carries them.
func newWireSettingsViews(views []setting.PluginSettingsView) []*v1.PluginSettingsView {
	out := make([]*v1.PluginSettingsView, 0, len(views))
	for _, one := range views {
		out = append(out, &v1.PluginSettingsView{Id: one.ID, Title: one.Title, After: one.After})
	}
	return out
}

// newWireReaders is the readers of a plugin as the wire carries them.
func newWireReaders(readers []setting.PluginReaderInfo) []*v1.PluginReader {
	out := make([]*v1.PluginReader, 0, len(readers))
	for _, one := range readers {
		out = append(out, &v1.PluginReader{Id: one.ID, Extensions: one.Extensions})
	}
	return out
}
