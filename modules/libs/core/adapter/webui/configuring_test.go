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

	// A model reaches the window with the setting it is read from and what
	// choosing it writes, which is the whole of what the window does with one.
	models := said.Msg.GetModels()
	if len(models) == 0 {
		t.Fatal("no setting names a model")
	}
	for _, one := range models {
		if len(one.GetNamedAt()) == 0 || len(one.GetWrites()) == 0 {
			t.Errorf("%q is read from nowhere, or writes nothing", one.GetTitle())
		}
	}
}

// A setting written is answered by the next question, and the model chosen for
// the agent stands where the models said it would.
func TestASettingWrittenIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	_, err := f.client.ChooseSettings(t.Context(), connect.NewRequest(&v1.ChooseSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	said, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(said.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", said.Msg.GetWritten())
	}
}

// A value the settings could not be read out of again is the client's to
// correct, and nothing is written.
func TestAValueTheSettingsCannotHoldIsRefused(t *testing.T) {
	for _, one := range []struct {
		what  string
		at    []string
		value string
	}{
		{"is not JSON at all", []string{"agent", "claude", "model"}, `opus`},
		{"is of the wrong shape", []string{"agent", "claude", "model"}, `7`},
		{"is past what the setting goes to", []string{"appearance", "text_scale"}, `5`},
	} {
		t.Run(one.what, func(t *testing.T) {
			f := opening(t, nil, nil, true)
			was, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
			if err != nil {
				t.Fatal(err)
			}

			_, err = f.client.ChooseSettings(
				t.Context(),
				connect.NewRequest(&v1.ChooseSettingsRequest{
					Settings: []*v1.Setting{{At: one.at, Value: one.value}},
				}),
			)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("the value was taken as %v", err)
			}

			// The settings still read, and read as they did.
			now, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
			if err != nil {
				t.Fatal(err)
			}
			if now.Msg.GetWritten() != was.Msg.GetWritten() {
				t.Errorf("the settings read out as %s", now.Msg.GetWritten())
			}
		})
	}
}

// Settings are written together or not at all, so one the settings cannot hold
// leaves the ones beside it where they were.
func TestSettingsWrittenTogetherLeaveTheFileAloneWhereOneIsRefused(t *testing.T) {
	f := opening(t, nil, nil, true)

	_, err := f.client.ChooseSettings(t.Context(), connect.NewRequest(&v1.ChooseSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
			{At: []string{"appearance", "text_scale"}, Value: `5`},
		},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("the settings were taken as %v", err)
	}

	said, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(said.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", said.Msg.GetWritten())
	}
}
