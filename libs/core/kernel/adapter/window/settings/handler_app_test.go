package settings

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	setting "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func TestGetSettingsAnswersTheDeclarationOfTheShell(t *testing.T) {
	handler := NewHandler(NewValues(setting.NewGet(
		func() (settingsdomain.Snapshot, error) {
			return settingsdomain.Snapshot{Content: `{"v":1}`, Path: "/home/numen.json", Version: 1}, nil
		},
		nil,
		func() []plugin.Setting {
			return []plugin.Setting{
				{Key: "appearance.interfaceScale", Type: "range", DefaultJSON: "1", ValueJSON: "1"},
			}
		},
		func() []manifest.SettingGroup { return []manifest.SettingGroup{{ID: "look", Title: "Look"}} },
	), nil, nil), numenv1connect.UnimplementedSettingsServiceHandler{})

	response, err := handler.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	app := response.Msg.GetApp()
	if len(app) != 1 || app[0].GetKey() != "appearance.interfaceScale" || app[0].GetDefaultJson() != "1" {
		t.Errorf("the declaration of the shell is %v", app)
	}
	groups := response.Msg.GetAppGroups()
	if len(groups) != 1 || groups[0].GetId() != "look" || groups[0].GetTitle() != "Look" {
		t.Errorf("the groups of the shell are %v", groups)
	}
}
