package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleConfigure(t *testing.T) {
	proof := &proofreadingServer{modelName: "initial-model"}
	life := server.NewLifecycleServer(
		make(chan struct{}),
		server.WithLifecycleUpdateConfig(proof.updateConfig),
	)

	req := connect.NewRequest(&numenv1.ConfigureRequest{
		Values: map[string]string{
			"model": "anthropic/claude-3-haiku",
		},
	})
	resp, err := life.Configure(context.Background(), req)
	if err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if !resp.Msg.GetSuccess() {
		t.Errorf("Configure success = false, want true")
	}

	info, err := proof.GetProofreaderInfo(context.Background(), connect.NewRequest(&numenv1.GetProofreaderInfoRequest{}))
	if err != nil {
		t.Fatalf("GetProofreaderInfo failed: %v", err)
	}
	if info.Msg.GetName() != "anthropic/claude-3-haiku" {
		t.Errorf("model = %q, want %q", info.Msg.GetName(), "anthropic/claude-3-haiku")
	}
}

func TestUpdateConfigReadsEverySettingTheManifestDeclares(t *testing.T) {
	proof := &proofreadingServer{}
	proof.updateConfig(map[string]string{
		"model":          "m",
		"baseUrl":        "http://example.test/v1",
		"strictMode":     "true",
		"maxSuggestions": "3",
	})
	if proof.modelName != "m" || proof.baseURL != "http://example.test/v1" || !proof.strict || proof.maxSuggestions != 3 {
		t.Errorf("config = %+v", proof)
	}
	proof.updateConfig(map[string]string{"strictMode": "false", "maxSuggestions": "bad"})
	if proof.strict || proof.maxSuggestions != 3 {
		t.Errorf("config after update = %+v", proof)
	}
}

func TestProofreadAppliesTheSettingsToTheModelCall(t *testing.T) {
	var system string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		system = body.Messages[0].Content
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{
			"role": "assistant", "content": "1|the cat sat on the mat\n2|the dog sat on the mat",
		}}}})
	}))
	defer api.Close()
	t.Setenv("NUMEN_PROOFREADING_KEY", "k")

	proof := &proofreadingServer{}
	proof.updateConfig(map[string]string{"baseUrl": api.URL, "strictMode": "true", "maxSuggestions": "1"})
	prompt := Opens + "1" + Closes + "the cta sat on the mat " + Opens + "2" + Closes + "the dgo sat on the mat"
	resp, err := proof.Proofread(context.Background(), connect.NewRequest(&numenv1.ProofreadRequest{
		Instruction: "base",
		Batches:     []*numenv1.ProofreadBatch{{Number: 0, Prompt: prompt}},
	}))
	if err != nil {
		t.Fatalf("Proofread failed: %v", err)
	}
	if got := resp.Msg.GetReplies()[0]; got != "1|the cat sat on the mat" {
		t.Errorf("reply = %q", got)
	}
	if !strings.HasPrefix(system, "base") || system == "base" {
		t.Errorf("instruction = %q, want base plus the strict rule", system)
	}
}

func newModelAPI(t *testing.T, content string, system *string) *httptest.Server {
	t.Helper()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		*system = body.Messages[0].Content
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": map[string]string{
			"role": "assistant", "content": content,
		}}}})
	}))
	t.Cleanup(api.Close)
	return api
}

func TestProofreadTextAnswersAScanWithTheSettingsApplied(t *testing.T) {
	var system string
	api := newModelAPI(t, "1|the cat sat on the mat\n2|the dog sat on the mat", &system)
	t.Setenv("NUMEN_PROOFREADING_KEY", "k")

	proof := &proofreadingServer{}
	proof.updateConfig(map[string]string{"baseUrl": api.URL, "strictMode": "true", "maxSuggestions": "1"})
	resp, err := proof.ProofreadText(context.Background(), connect.NewRequest(&numenv1.ProofreadTextRequest{
		Kind: numenv1.ProofreadKind_PROOFREAD_KIND_SCAN,
		Groups: []*numenv1.ProofreadGroup{{Number: 0, Lines: []*numenv1.ProofreadLine{
			{Number: 1, Text: "the cta sat on the mat"},
			{Number: 2, Text: "the dgo sat on the mat"},
		}}},
	}))
	if err != nil {
		t.Fatalf("ProofreadText failed: %v", err)
	}
	got := resp.Msg.GetCorrections()
	if len(got) != 1 || got[0].GetNumber() != 1 || got[0].GetText() != "the cat sat on the mat" {
		t.Errorf("corrections = %v", got)
	}
	if !resp.Msg.GetIsDone() || resp.Msg.GetUncorrected() != 0 {
		t.Errorf("response = %v", resp.Msg)
	}
	if !strings.HasPrefix(system, scanInstruction) || system == scanInstruction {
		t.Errorf("instruction = %q, want the scan instruction plus the strict rule", system)
	}
}

func TestProofreadTextAnswersSpeechAWindowAtATime(t *testing.T) {
	var system string
	api := newModelAPI(t, "", &system)
	t.Setenv("NUMEN_PROOFREADING_KEY", "k")

	proof := &proofreadingServer{}
	proof.updateConfig(map[string]string{"baseUrl": api.URL})
	var lines []*numenv1.ProofreadLine
	for n := int32(0); n < 10; n++ {
		lines = append(lines, &numenv1.ProofreadLine{Number: n, Text: "some words"})
	}
	resp, err := proof.ProofreadText(context.Background(), connect.NewRequest(&numenv1.ProofreadTextRequest{
		Kind:      numenv1.ProofreadKind_PROOFREAD_KIND_SPEECH,
		Groups:    []*numenv1.ProofreadGroup{{Lines: lines}},
		Size:      4,
		Overlap:   1,
		LineLimit: 6,
	}))
	if err != nil {
		t.Fatalf("ProofreadText failed: %v", err)
	}
	if resp.Msg.GetIsDone() || resp.Msg.GetNextLine() != 5 {
		t.Errorf("response = %v, want to continue from line 5", resp.Msg)
	}
	if system != speechInstruction {
		t.Errorf("instruction = %q", system)
	}
}

func TestProofreadTextRefusesWhatItCannotAnswer(t *testing.T) {
	proof := &proofreadingServer{}
	t.Setenv("NUMEN_PROOFREADING_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	_, err := proof.ProofreadText(context.Background(), connect.NewRequest(&numenv1.ProofreadTextRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("an unspecified kind gave %v", err)
	}
	_, err = proof.ProofreadText(context.Background(), connect.NewRequest(&numenv1.ProofreadTextRequest{
		Kind: numenv1.ProofreadKind_PROOFREAD_KIND_SCAN,
	}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("no key gave %v", err)
	}
}

// Every kind of text the schema names is one ProofreadText answers.
func TestProofreadTextAnswersEveryKindTheSchemaNames(t *testing.T) {
	t.Setenv("NUMEN_PROOFREADING_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	proof := &proofreadingServer{}

	values := numenv1.ProofreadKind(0).Descriptor().Values()
	for i := range values.Len() {
		kind := numenv1.ProofreadKind(values.Get(i).Number())
		if kind == numenv1.ProofreadKind_PROOFREAD_KIND_UNSPECIFIED {
			continue
		}
		_, err := proof.ProofreadText(context.Background(), connect.NewRequest(&numenv1.ProofreadTextRequest{Kind: kind}))
		if connect.CodeOf(err) == connect.CodeInvalidArgument {
			t.Errorf("%s was refused as unknown", kind)
		}
	}
}
