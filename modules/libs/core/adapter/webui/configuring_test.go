package webui_test

import (
	"encoding/json"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The window asks for every setting and writes one of them back.

// setting is what stands at a path through the settings, read off the whole of
// them. What the file leaves out stands there at its default.
func setting(t *testing.T, f *going, at ...string) any {
	t.Helper()
	said, err := f.client.Settings(t.Context(), connect.NewRequest(&v1.SettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var held any
	if err := json.Unmarshal([]byte(said.Msg.GetWritten()), &held); err != nil {
		t.Fatalf("the settings read out as %q: %v", said.Msg.GetWritten(), err)
	}
	for _, step := range at {
		section, is := held.(map[string]any)
		if !is {
			t.Fatalf("%s is not a section of the settings", strings.Join(at, "."))
		}
		held = section[step]
	}
	return held
}

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

// The window opens the settings file whole, and what it reads is the file as
// its person wrote it.
func TestTheSettingsFileIsReadAsItStands(t *testing.T) {
	f := opening(t, nil, nil, true)

	said, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetPath() == "" {
		t.Error("the file stands nowhere")
	}

	var held map[string]any
	if err := json.Unmarshal([]byte(said.Msg.GetWritten()), &held); err != nil {
		t.Fatalf("the file reads as %q: %v", said.Msg.GetWritten(), err)
	}
}

// The file is written as it was typed, and read back the same way.
func TestTheSettingsFileIsWrittenAsItWasTyped(t *testing.T) {
	f := opening(t, nil, nil, true)
	written := "{\n  \"agent\": {\n    \"claude\": { \"model\": \"opus\" }\n  }\n}\n"

	if _, err := f.client.WriteSettingsFile(
		t.Context(),
		connect.NewRequest(&v1.WriteSettingsFileRequest{Written: written}),
	); err != nil {
		t.Fatal(err)
	}

	said, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetWritten() != written {
		t.Errorf("the file reads as %q, wanted %q", said.Msg.GetWritten(), written)
	}
}

// A file the settings cannot be read out of is the client's to correct, and
// what a person already has in the file is worth more than the write.
func TestAFileTheSettingsCannotBeReadOutOfIsRefused(t *testing.T) {
	f := opening(t, nil, nil, true)
	was, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range []struct {
		name    string
		written string
	}{
		{"not JSON at all", `{ "agent": `},
		{"a value of the wrong kind", `{"appearance": {"text_scale": "large"}}`},
		{"a number past what its setting goes to", `{"appearance": {"text_scale": 5}}`},
	} {
		t.Run(one.name, func(t *testing.T) {
			_, err := f.client.WriteSettingsFile(
				t.Context(),
				connect.NewRequest(&v1.WriteSettingsFileRequest{Written: one.written}),
			)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("the file was taken as %v", err)
			}

			now, err := f.client.SettingsFile(
				t.Context(),
				connect.NewRequest(&v1.SettingsFileRequest{}),
			)
			if err != nil {
				t.Fatal(err)
			}
			if now.Msg.GetWritten() != was.Msg.GetWritten() {
				t.Errorf("the file reads as %q", now.Msg.GetWritten())
			}
		})
	}
}

// The file holds a person's keys, and what a refusal says does not repeat them.
func TestWhatARefusalSaysDoesNotRepeatWhatStandsInTheFile(t *testing.T) {
	f := opening(t, nil, nil, true)
	secret := "sk-not-a-real-key-0000"

	_, err := f.client.WriteSettingsFile(
		t.Context(),
		connect.NewRequest(&v1.WriteSettingsFileRequest{
			Written: `{"appearance": {"text_scale": "` + secret + `"}}`,
		}),
	)
	if err == nil {
		t.Fatal("the file was taken")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("the refusal says %q", err)
	}
}

// presented is the file a client says it last read.
func presented(written string) *string { return &written }

// The settings page and the file's own tab both write this file. A tab
// presenting a file the settings page has since patched is answered the
// question, and the patch stands.
func TestAFileThatMovedPastWhatTheClientReadIsAnswered(t *testing.T) {
	f := opening(t, nil, nil, true)
	was, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.client.ChooseSettings(t.Context(), connect.NewRequest(&v1.ChooseSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
		},
	})); err != nil {
		t.Fatal(err)
	}
	patched, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	said, err := f.client.WriteSettingsFile(t.Context(), connect.NewRequest(
		&v1.WriteSettingsFileRequest{
			Written: "{\n  \"agent\": { \"use\": \"claude\" }\n}\n",
			Seen:    presented(was.Msg.GetWritten()),
		},
	))
	if err != nil {
		t.Fatal(err)
	}
	if !said.Msg.GetChanged() {
		t.Error("the write landed, wanted the question put to the person")
	}

	now, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if now.Msg.GetWritten() != patched.Msg.GetWritten() {
		t.Errorf("the file reads as %q, wanted %q", now.Msg.GetWritten(), patched.Msg.GetWritten())
	}
}

// A client presenting the file it read writes over it.
func TestAFileStandingAtWhatTheClientReadIsWritten(t *testing.T) {
	f := opening(t, nil, nil, true)
	was, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	written := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"

	said, err := f.client.WriteSettingsFile(t.Context(), connect.NewRequest(
		&v1.WriteSettingsFileRequest{Written: written, Seen: presented(was.Msg.GetWritten())},
	))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetChanged() {
		t.Fatal("the write was answered the question, wanted it to land")
	}

	now, err := f.client.SettingsFile(t.Context(), connect.NewRequest(&v1.SettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if now.Msg.GetWritten() != written {
		t.Errorf("the file reads as %q, wanted %q", now.Msg.GetWritten(), written)
	}
}
