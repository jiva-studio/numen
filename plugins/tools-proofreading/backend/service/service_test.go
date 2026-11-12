package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/ai"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/internal/adapter/model"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/usecase/proofread"
)

const (
	ocrID    = "md.test.alpha"
	scanPath = "books/flywheel.pdf"
)

// scene is a vault with a scan the OCR plugin made a text of, and the proofreading over it as the
// plugin mounts it.
type scene struct {
	texts     *hosttest.Texts
	host      proofread.Host
	settings  *settings.Store
	questions []*ai.Question
	script    func(prompt string) string
	run       *proofread.Proofreading
}

func newScene(t *testing.T) *scene {
	t.Helper()
	vault := hosttest.New("Physics")
	s := &scene{
		texts:    hosttest.NewTexts("Physics", PluginID),
		settings: settings.NewStore(),
		script:   func(string) string { return "" },
	}
	s.texts.File(scanPath, []byte("the scan"))
	store := vault.Store()
	s.texts.SetStore(PluginID, store)
	complete := func(_ context.Context, question *ai.Question) (string, error) {
		s.questions = append(s.questions, question)
		return s.script(question.GetMessages()[1].GetText()), nil
	}
	s.host = proofread.Host{
		Store: func(id domain.VaultID) host.StoreWriter { return store.With(id) },
		Texts: func(id domain.VaultID) proofread.Texts { return s.texts.With(id) },
		Queue: hosttest.NewQueue(),
	}
	s.run = newProofreading(s.host, complete, s.settings)
	return s
}

func (s *scene) holdScan(t *testing.T) domain.TextVersion {
	t.Helper()
	parts := []domain.TextSegment{
		{Text: "A flywheel keeps energy ", Selector: `{"page":1}`},
		{Text: "in a rotor spinnig in a vacum.\n\n", Selector: `{"page":1}`},
		{Text: "Pumped storage lifts water.\n\n", Selector: `{"page":2}`},
	}
	if err := host.WriteText(t.Context(), s.texts.GetStore(ocrID), "recognition.jsonl", parts); err != nil {
		t.Fatal(err)
	}
	version, err := s.texts.As(ocrID).PutVersion(t.Context(), host.TextPut{
		Path: scanPath, Key: "recognition", Kind: "recognition", InputHash: domain.HashContent([]byte("the scan")), File: "recognition.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func (s *scene) readLiveSegments(t *testing.T) ([]domain.TextSegment, domain.TextVersion) {
	t.Helper()
	listing, err := s.texts.ListVersions(t.Context(), scanPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.GetID() == listing.Live {
			parts, err := proofread.ReadParts(t.Context(), s.texts, version)
			if err != nil {
				t.Fatal(err)
			}
			return parts, version
		}
	}
	t.Fatalf("%s has no live version: %+v", scanPath, listing)
	return nil, domain.TextVersion{}
}

func getRows(text string, replace func(text string) string) string {
	var rows []string
	for _, segment := range strings.Split(text, numbered.Opens)[1:] {
		number, line, _ := strings.Cut(segment, numbered.Closes)
		line = strings.TrimSpace(line)
		if fixed := replace(line); fixed != line {
			rows = append(rows, number+"|"+fixed)
		}
	}
	return strings.Join(rows, "\n")
}

func TestProofreadWithoutAVersionUsesTheLiveOne(t *testing.T) {
	s := newScene(t)
	s.holdScan(t)
	s.script = func(text string) string {
		return getRows(text, func(line string) string { return strings.ReplaceAll(line, "vacum", "vacuum") })
	}

	if _, err := newProofreadCommand(s.run)(t.Context(), command.Request{Command: "proofreading.create", Path: scanPath}); err != nil {
		t.Fatal(err)
	}

	if _, live := s.readLiveSegments(t); live.PluginID != PluginID {
		t.Errorf("the live version is %+v", live)
	}
}

func TestThePluginRunsNeitherDeleteTextNorTheCommandOfBefore(t *testing.T) {
	s := newScene(t)
	commands, err := newCommands(s.run)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "app:version.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("Delete text, which the core runs, answered %v, want ErrUnknownCommand", err)
	}
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "proofreading.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("the command of before answered %v, want ErrUnknownCommand", err)
	}
}

func TestTheMockMarksTheLinesTheGatesLetThrough(t *testing.T) {
	s := newScene(t)
	scan := s.holdScan(t)
	s.run = newProofreading(s.host, model.CompleteMock, s.settings)

	if _, err := s.run.Proofread(t.Context(), proofread.Request{Path: scanPath, Source: scan.GetID()}); err != nil {
		t.Fatal(err)
	}

	parts, _ := s.readLiveSegments(t)
	if parts[0].Text != "A flywheel keeps energy " {
		t.Errorf("the gate let a mark on a short line through: %q", parts[0].Text)
	}
	for _, part := range parts[1:] {
		if !strings.HasSuffix(strings.TrimSpace(part.Text), " "+model.MockMark) {
			t.Errorf("the mock left %q", part.Text)
		}
	}
}

func TestProofreadUsesDirectChatAPIWhenProviderIsConfigured(t *testing.T) {
	var gotAuth, gotModel, gotSystem string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) != 2 {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		gotModel, gotSystem = req.Model, req.Messages[0].Content
		reply := getRows(req.Messages[1].Content, func(line string) string {
			return strings.NewReplacer("spinnig", "spinning", "vacum", "vacuum").Replace(line)
		})
		if err := json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": reply}}},
		}); err != nil {
			t.Errorf("encode the response: %v", err)
		}
	}))
	defer srv.Close()

	s := newScene(t)
	s.settings.Update(map[string]string{
		"provider": "custom",
		"baseUrl":  srv.URL,
		"model":    "google/gemini-2.5-flash",
	}, server.Secrets{server.SecretAPIKey: {
		{ID: "s00000001", Name: "Work", Group: "", Value: "secret-key"},
		{ID: "s00000002", Name: "Home", Group: "", Value: "other-key"},
	}})
	scan := s.holdScan(t)

	res, err := s.run.Proofread(t.Context(), proofread.Request{Path: scanPath, Source: scan.GetID()})
	if err != nil {
		t.Fatal(err)
	}
	if res.Corrected != 1 {
		t.Errorf("corrected = %d, want 1", res.Corrected)
	}
	if len(s.questions) != 0 {
		t.Errorf("agent completer was called %d times, want 0", len(s.questions))
	}
	if gotAuth != "Bearer secret-key" {
		t.Errorf("Authorization = %q, want Bearer secret-key", gotAuth)
	}
	if gotModel != "google/gemini-2.5-flash" {
		t.Errorf("model = %q, want google/gemini-2.5-flash", gotModel)
	}
	if gotSystem != numbered.ScanInstruction {
		t.Errorf("system instruction mismatch")
	}
}
