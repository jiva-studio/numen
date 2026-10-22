package editor_test

import (
	"encoding/json"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// The window asks for every setting and writes one of them back.

// them. What the file leaves out stands there at its default.
func getSetting(t *testing.T, f *going, pos ...string) any {
	t.Helper()
	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var target any
	if err := json.Unmarshal([]byte(response.Msg.GetWritten()), &target); err != nil {
		t.Fatalf("the settings read out as %q: %v", response.Msg.GetWritten(), err)
	}
	for _, step := range pos {
		section, is := target.(map[string]any)
		if !is {
			t.Fatalf("%s is not a section of the settings", strings.Join(pos, "."))
		}
		target = section[step]
	}
	return target
}

// The settings read out hold what the file leaves out, so the window draws what
// this installation is doing.
func TestTheSettingsReadOutStandOnTheDefaults(t *testing.T) {
	f := opening(t, nil, nil, true)

	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetPath() == "" {
		t.Error("the settings stand nowhere")
	}

	var target map[string]any
	if err := json.Unmarshal([]byte(response.Msg.GetWritten()), &target); err != nil {
		t.Fatalf("the settings read out as %q: %v", response.Msg.GetWritten(), err)
	}
	if _, exists := target["appearance"]; !exists {
		t.Errorf("the settings read out as %v", target)
	}

	// A model reaches the window with the setting it is read from and what
	// choosing it writes, which is the whole of what the window does with one.
	models := response.Msg.GetModels()
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
func TestASettingSavedIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", response.Msg.GetWritten())
	}
}

// A value the settings could not be read out of again is the client's to
// correct, and nothing is written.
func TestAValueTheSettingsCannotHoldIsRefused(t *testing.T) {
	for _, one := range []struct {
		what   string
		posVal []string
		value  string
	}{
		{"is not JSON at all", []string{"agent", "claude", "model"}, `opus`},
		{"is of the wrong shape", []string{"agent", "claude", "model"}, `7`},
		{"is past what the setting goes to", []string{"appearance", "text_scale"}, `5`},
	} {
		t.Run(one.what, func(t *testing.T) {
			f := opening(t, nil, nil, true)
			was, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
			if err != nil {
				t.Fatal(err)
			}

			_, err = f.configuring.WriteSettings(
				t.Context(),
				connect.NewRequest(&v1.WriteSettingsRequest{
					Settings: []*v1.Setting{{At: one.posVal, Value: one.value}},
				}),
			)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("the value was taken as %v", err)
			}

			// The settings still read, and read as they did.
			now, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
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
func TestSettingsSavedTogetherLeaveTheFileAloneWhereOneIsRefused(t *testing.T) {
	f := opening(t, nil, nil, true)

	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
			{At: []string{"appearance", "text_scale"}, Value: `5`},
		},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("the settings were taken as %v", err)
	}

	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", response.Msg.GetWritten())
	}
}

// The window opens the settings file whole, and what it reads is the file as
// its person wrote it.
func TestTheSettingsFileIsReadAsItExists(t *testing.T) {
	f := opening(t, nil, nil, true)

	response, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetPath() == "" {
		t.Error("the file stands nowhere")
	}

	var target map[string]any
	if err := json.Unmarshal([]byte(response.Msg.GetWritten()), &target); err != nil {
		t.Fatalf("the file reads as %q: %v", response.Msg.GetWritten(), err)
	}
}

// The file is written as it was typed, and read back the same way.
func TestTheSettingsFileIsSavedAsItWasTyped(t *testing.T) {
	f := opening(t, nil, nil, true)
	content := "{\n  \"agent\": {\n    \"claude\": { \"model\": \"opus\" }\n  }\n}\n"

	if _, err := f.configuring.WriteSettingsFile(
		t.Context(),
		connect.NewRequest(&v1.WriteSettingsFileRequest{Written: content}),
	); err != nil {
		t.Fatal(err)
	}

	response, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetWritten() != content {
		t.Errorf("the file reads as %q, wanted %q", response.Msg.GetWritten(), content)
	}
}

// A file the settings cannot be read out of is the client's to correct, and
// what a person already has in the file is worth more than the write.
func TestAFileTheSettingsCannotBeReadOutOfIsRefused(t *testing.T) {
	f := opening(t, nil, nil, true)
	was, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range []struct {
		name    string
		encoded string
	}{
		{"not JSON at all", `{ "agent": `},
		{"a value of the wrong kind", `{"appearance": {"text_scale": "large"}}`},
		{"a number past what its setting goes to", `{"appearance": {"text_scale": 5}}`},
	} {
		t.Run(one.name, func(t *testing.T) {
			_, err := f.configuring.WriteSettingsFile(
				t.Context(),
				connect.NewRequest(&v1.WriteSettingsFileRequest{Written: one.encoded}),
			)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("the file was taken as %v", err)
			}

			now, err := f.configuring.ReadSettingsFile(
				t.Context(),
				connect.NewRequest(&v1.ReadSettingsFileRequest{}),
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

// The file holds a person's keys, and what an error says does not repeat them.
func TestWhatAnErrorReturnsDoesNotRepeatWhatExistsInTheFile(t *testing.T) {
	f := opening(t, nil, nil, true)
	secret := "sk-not-a-real-key-0000"

	_, err := f.configuring.WriteSettingsFile(
		t.Context(),
		connect.NewRequest(&v1.WriteSettingsFileRequest{
			Written: `{"appearance": {"text_scale": "` + secret + `"}}`,
		}),
	)
	if err == nil {
		t.Fatal("the file was taken")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("the error says %q", err)
	}
}

// newString is the file a client says it last read.
func newString(output string) *string { return &output }

// The settings page and the file's own tab both write this file. A tab
// presenting a file the settings page has since patched is answered the
// question, and the patch stands.
func TestAFileThatMovedPastWhatTheClientReadIsAnswered(t *testing.T) {
	f := opening(t, nil, nil, true)
	was, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
		},
	})); err != nil {
		t.Fatal(err)
	}
	patched, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	response, err := f.configuring.WriteSettingsFile(t.Context(), connect.NewRequest(
		&v1.WriteSettingsFileRequest{
			Written:  "{\n  \"agent\": { \"use\": \"claude\" }\n}\n",
			Baseline: newString(was.Msg.GetWritten()),
		},
	))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError() != v1.ErrorCode_ERROR_CODE_STALE {
		t.Error("the write landed, wanted the question put to the person")
	}

	now, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if now.Msg.GetWritten() != patched.Msg.GetWritten() {
		t.Errorf("the file reads as %q, wanted %q", now.Msg.GetWritten(), patched.Msg.GetWritten())
	}
}

// A client presenting the file it read writes over it.
func TestAFileActiveAtWhatTheClientReadIsSaved(t *testing.T) {
	f := opening(t, nil, nil, true)
	was, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	content := "{\n  \"agent\": { \"use\": \"claude\" }\n}\n"

	response, err := f.configuring.WriteSettingsFile(t.Context(), connect.NewRequest(
		&v1.WriteSettingsFileRequest{Written: content, Baseline: newString(was.Msg.GetWritten())},
	))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatal("the write was answered the question, wanted it to land")
	}

	now, err := f.configuring.ReadSettingsFile(t.Context(), connect.NewRequest(&v1.ReadSettingsFileRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if now.Msg.GetWritten() != content {
		t.Errorf("the file reads as %q, wanted %q", now.Msg.GetWritten(), content)
	}
}

// The levels of reasoning a model offers reach the window with the model.
func TestTheLevelsOfAModelReachTheWindow(t *testing.T) {
	got := editor.NewWireModels([]settings.Model{{
		Name: "deep", ReasoningLevels: []settings.ReasoningLevel{{Name: "low", Title: "Low"}}, DefaultReasoning: "low",
	}, {Name: "plain"}})

	if levels := got[0].GetReasoningLevels(); len(levels) != 1 || levels[0].GetName() != "low" || levels[0].GetTitle() != "Low" ||
		got[0].GetDefaultReasoning() != "low" {
		t.Errorf("the model is %v", got[0])
	}
	if len(got[1].GetReasoningLevels()) != 0 {
		t.Errorf("a model offering none is %v", got[1])
	}
}
