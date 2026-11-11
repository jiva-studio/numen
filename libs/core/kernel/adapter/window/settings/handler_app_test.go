package settings

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	setting "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func TestGetSettingsAnswersTheDeclarationOfTheShell(t *testing.T) {
	handler := NewHandler(NewValues(setting.NewGetWithSecrets(setting.NewGet(
		func() (string, string, error) { return `{"v":1}`, "/home/numen.json", nil },
		nil, nil,
		func() []plugin.Setting {
			return []plugin.Setting{
				{Key: "appearance.interfaceScale", Type: "range", DefaultJSON: "1", ValueJSON: "1"},
			}
		},
	), nil), nil, nil, nil), PluginFolders{}, SecretItems{}, SecretGroups{})

	response, err := handler.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	app := response.Msg.GetApp()
	if len(app) != 1 || app[0].GetKey() != "appearance.interfaceScale" || app[0].GetDefaultJson() != "1" {
		t.Errorf("the declaration of the shell is %v", app)
	}
}
