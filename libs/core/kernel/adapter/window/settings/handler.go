// Package settings provides settings RPC handlers for the editor window.
package settings

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	setting "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// Handler answers every setting read and written. Each
// part holds the use cases of one kind of call, so no value holds more than five.
type Handler struct {
	Values
	PluginFolders
}

// Values answers the reads and writes of the settings and the choice of a path. A value with no
// use case set answers that the build serves no settings.
type Values struct {
	get        *setting.Get
	write      *setting.WriteSettings
	watch      *setting.WatchSettingChanges
	choosePath *setting.ChoosePluginPath
}

func NewValues(
	get *setting.Get, write *setting.WriteSettings, watch *setting.WatchSettingChanges,
	choosePath *setting.ChoosePluginPath,
) Values {
	return Values{get: get, write: write, watch: watch, choosePath: choosePath}
}

// NewHandler is the handler of SettingsService made of its parts.
func NewHandler(values Values, folders PluginFolders) *Handler {
	return &Handler{Values: values, PluginFolders: folders}
}

// PluginFolders answers the calls about the folders plugins keep in the shown vault and the start of
// a plugin that failed to start.
type PluginFolders struct {
	retry  *setting.RetryPlugin
	data   PluginData
	vaults VaultProvider
}

// NewPluginFolders is the part of SettingsService that answers about the folders of plugins in the shown vault.
func NewPluginFolders(retry *setting.RetryPlugin, data PluginData, vaults VaultProvider) PluginFolders {
	return PluginFolders{retry: retry, data: data, vaults: vaults}
}

// errNoSettings is what a build that serves no settings answers.
var errNoSettings = errors.New("this build serves no settings")

// GetSettings returns every setting of the file this installation is configured in.
func (h *Values) GetSettings(
	_ context.Context, _ *connect.Request[v1.GetSettingsRequest],
) (*connect.Response[v1.GetSettingsResponse], error) {
	if h.get == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	got, err := h.get.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	reply := &v1.GetSettingsResponse{
		Written: got.Written,
		Path:    got.Path,
		Plugins: NewWirePluginInfos(got.Plugins),
		App:     newWirePluginSettings(got.App),
	}
	return connect.NewResponse(reply), nil
}

// WriteSettings writes settings into that file. A value the settings could not
// be read out of again is the client's to correct, and the file is left as it
// was.
func (h *Values) WriteSettings(
	_ context.Context, r *connect.Request[v1.WriteSettingsRequest],
) (*connect.Response[v1.WriteSettingsResponse], error) {
	if h.write == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	content := make([]settingsdomain.Setting, 0, len(r.Msg.GetSettings()))
	for _, one := range r.Msg.GetSettings() {
		content = append(content, settingsdomain.Setting{Path: one.GetAt(), JSON: one.GetValue()})
	}
	if err := h.write.Execute(content); err != nil {
		if errors.Is(err, settingsdomain.ErrNotASetting) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteSettingsResponse{}), nil
}

func newWireSettings(target []settingsdomain.Setting) []*v1.Setting {
	content := make([]*v1.Setting, 0, len(target))
	for _, one := range target {
		content = append(content, &v1.Setting{At: one.Path, Value: one.JSON})
	}
	return content
}

// NewWirePluginInfos is the discovered plugins and their declarative settings, as the wire carries it.
func NewWirePluginInfos(target []plugin.Info) []*v1.PluginInfo {
	infos := make([]*v1.PluginInfo, 0, len(target))
	for _, one := range target {
		infos = append(infos, &v1.PluginInfo{
			Id:             one.ID,
			Name:           one.Name,
			Version:        one.Version,
			Description:    one.Description,
			Capabilities:   one.Capabilities,
			Settings:       newWirePluginSettings(one.Settings),
			IsActive:       one.IsActive,
			Permissions:    one.Permissions,
			HasFrontend:    one.HasFrontend,
			HasBackend:     one.HasBackend,
			Readers:        newWireReaders(one.Readers),
			SettingsViews:  newWireSettingsViews(one.SettingsViews),
			Contributions:  newWireContributions(one.Contributions),
			IsOffByDefault: one.IsOffByDefault,
		})
	}
	return infos
}

func newWirePluginSettings(target []plugin.Setting) []*v1.PluginSetting {
	settings := make([]*v1.PluginSetting, 0, len(target))
	for _, s := range target {
		settings = append(settings, &v1.PluginSetting{
			Key:           s.Key,
			Type:          s.Type,
			Title:         s.Title,
			Description:   s.Description,
			DefaultJson:   s.DefaultJSON,
			Palette:       s.Palette,
			Icon:          s.Icon,
			Options:       newWireOptions(s.Options),
			Min:           s.Min,
			Max:           s.Max,
			Step:          s.Step,
			OptionsBy:     newWireOptionsBy(s.OptionsBy),
			When:          newWireCondition(s.When),
			Accept:        s.Accept,
			Multiple:      s.Multiple,
			Preview:       s.Preview,
			OptionsFrom:   s.OptionsFrom,
			Operations:    s.Operations,
			ActiveOptions: newWireOptions(s.ActiveOptions),
			IsShown:       s.IsShown,
			ValueJson:     s.ValueJSON,
		})
	}
	return settings
}

// RetryPlugin starts a plugin whose last start failed once more. A plugin that fails
// again is answered with its reason, and one the host does not know is not found.
func (h *PluginFolders) RetryPlugin(
	ctx context.Context, r *connect.Request[v1.RetryPluginRequest],
) (*connect.Response[v1.RetryPluginResponse], error) {
	if h.retry == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	failure, err := h.retry.Execute(ctx, r.Msg.GetPluginId())
	if err != nil {
		var unavailable *plugin.UnavailableError
		if errors.As(err, &unavailable) && unavailable.Reason == plugin.NotInstalled {
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

func newWireOptions(target manifest.SettingOptions) []*v1.PluginSettingOption {
	if len(target) == 0 {
		return nil
	}
	options := make([]*v1.PluginSettingOption, 0, len(target))
	for _, one := range target {
		options = append(options, newWireOption(one))
	}
	return options
}

func newWireOption(one manifest.SettingOption) *v1.PluginSettingOption {
	return &v1.PluginSettingOption{Value: one.Value, Title: one.Title, Detail: one.Detail, When: one.When}
}

func newWireOptionsBy(target *manifest.OptionsBy) *v1.PluginSettingOptionsBy {
	if target == nil {
		return nil
	}
	options := make(map[string]*v1.PluginSettingOptions, len(target.Options))
	for value, list := range target.Options {
		options[value] = &v1.PluginSettingOptions{Options: newWireOptions(list)}
	}
	return &v1.PluginSettingOptionsBy{Setting: target.Setting, Options: options}
}

func newWireCondition(target *manifest.Condition) *v1.PluginSettingCondition {
	if target == nil {
		return nil
	}
	return &v1.PluginSettingCondition{Setting: target.Setting, Is: target.Is}
}

// SetPathChooser sets the native dialog a path setting is chosen with, once the application has one.
func (h *Values) SetPathChooser(choose *setting.ChoosePluginPath) { h.choosePath = choose }

// ChoosePluginPath shows the machine's native picker for a plugin's path setting
// and writes the chosen path. A path the setting does not accept returns the reason.
func (h *Values) ChoosePluginPath(
	ctx context.Context, r *connect.Request[v1.ChoosePluginPathRequest],
) (*connect.Response[v1.ChoosePluginPathResponse], error) {
	if h.choosePath == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	chosen, err := h.choosePath.Execute(ctx, r.Msg.GetPluginId(), r.Msg.GetKey())
	reply := &v1.ChoosePluginPathResponse{Path: chosen.Path, IsChosen: chosen.IsChosen}
	switch {
	case errors.Is(err, settingsdomain.ErrNotASetting):
		reply.Problem = strings.TrimPrefix(err.Error(), settingsdomain.ErrNotASetting.Error()+": ")
	case errors.Is(err, setting.ErrNoPathSetting):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case err != nil:
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(reply), nil
}

func newWireSettingsViews(views []manifest.SettingsView) []*v1.PluginSettingsView {
	out := make([]*v1.PluginSettingsView, 0, len(views))
	for _, one := range views {
		out = append(out, &v1.PluginSettingsView{Id: one.ID, Title: one.Title})
	}
	return out
}

func newWireReaders(readers []plugin.ReaderInfo) []*v1.PluginReader {
	out := make([]*v1.PluginReader, 0, len(readers))
	for _, one := range readers {
		out = append(out, &v1.PluginReader{Id: one.ID, Extensions: one.Extensions})
	}
	return out
}

// WatchSettings streams setting changes until the client disconnects.
func (h *Values) WatchSettings(
	ctx context.Context, _ *connect.Request[v1.WatchSettingsRequest], out *connect.ServerStream[v1.WatchSettingsResponse],
) error {
	if h.watch == nil {
		return connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	changes, stop := h.watch.Execute()
	defer stop()
	if err := out.Send(&v1.WatchSettingsResponse{}); err != nil {
		return err
	}
	repeat := time.NewTicker(keepalive)
	defer repeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			if err := out.Send(&v1.WatchSettingsResponse{}); err != nil {
				return err
			}
		case batch, open := <-changes:
			if !open {
				return nil
			}
			if err := out.Send(&v1.WatchSettingsResponse{Changed: newWireSettings(batch)}); err != nil {
				return err
			}
		}
	}
}

const keepalive = 30 * time.Second
