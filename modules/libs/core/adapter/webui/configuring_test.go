package webui_test

import (
	"encoding/json"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The window asks for every setting and writes one of them back.

// The settings read out hold what the file leaves out, so the window draws what
// this installation is doing rather than what a person happened to type.
func TestTheSettingsReadOutStandOnTheDefaults(t *testing.T) {
	f := opening(t, nil, nil, true)

	said, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetPath() == "" {
		t.Error("the settings stand nowhere")
	}

	var held map[string]any
	if err := json.Unmarshal([]byte(said.Msg.GetWritten()), &held); err != nil {
		t.Fatalf("the settings read out as %q: %v", said.Msg.GetWritten(), err)
	}
	if _, stands := held["appearance"]; !stands {
		t.Errorf("the settings read out as %v", held)
	}
}

// Every model says where it is read from and what choosing it writes, and one
// of the models a setting offers is the one this installation runs on.
func TestTheModelsOfferedSayWhereTheyAreWritten(t *testing.T) {
	f := opening(t, nil, nil, true)

	said, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	models := said.Msg.GetModels()
	if len(models) == 0 {
		t.Fatal("no setting names a model")
	}

	byDefault := map[string]int{}
	for _, one := range models {
		if len(one.GetNamedAt()) == 0 || len(one.GetWrites()) == 0 {
			t.Errorf("%q is read from nowhere, or writes nothing", one.GetTitle())
		}
		if one.GetByDefault() {
			byDefault[strings.Join(one.GetNamedAt(), ".")]++
		}
	}
	for setting, count := range byDefault {
		if count != 1 {
			t.Errorf("%s has %d models by default", setting, count)
		}
	}
}

// A setting written is answered by the next question, and the model chosen for
// the agent stands where the models said it would.
func TestASettingWrittenIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	turned, err := f.client.ChooseSetting(t.Context(), connect.NewRequest(&v1.ChooseSettingRequest{
		Settings: []*v1.Written{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := turned.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the setting was refused: %v", refusal)
	}

	said, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(said.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", said.Msg.GetWritten())
	}
}

// A value the file cannot hold is the client's to correct, and nothing is
// written.
func TestASettingThatIsNotJSONIsRefused(t *testing.T) {
	f := opening(t, nil, nil, true)

	_, err := f.client.ChooseSetting(t.Context(), connect.NewRequest(&v1.ChooseSettingRequest{
		Settings: []*v1.Written{{At: []string{"agent", "claude", "model"}, Value: `opus`}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("the value was taken as %v", err)
	}

	said, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(said.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", said.Msg.GetWritten())
	}
}
