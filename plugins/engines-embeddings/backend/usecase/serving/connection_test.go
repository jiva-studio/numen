package serving

import (
	"context"
	"errors"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

// hostModels simulates the model API of the host: it answers with a profile and embeds with the
// width of that profile, and it records what it was asked.
type hostModels struct {
	mu           sync.Mutex
	profile      model.ConnectionProfile
	profileErr   error
	embedErr     error
	width        int
	profileCalls int
	hold         chan struct{}
	embedded     []embedCall
}

type embedCall struct {
	role  model.Role
	texts []string
}

func (h *hostModels) GetEmbeddingProfile(context.Context) (model.ConnectionProfile, error) {
	h.mu.Lock()
	h.profileCalls++
	hold := h.hold
	h.mu.Unlock()
	if hold != nil {
		<-hold
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.profile, h.profileErr
}

func (h *hostModels) Embed(_ context.Context, role model.Role, texts []string) ([][]float32, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.embedded = append(h.embedded, embedCall{role: role, texts: texts})
	if h.embedErr != nil {
		return nil, h.embedErr
	}
	width := h.width
	if width == 0 {
		width = h.profile.Dimensions
	}
	vectors := make([][]float32, len(texts))
	for i := range vectors {
		vectors[i] = make([]float32, width)
		vectors[i][0] = float32(i + 1)
	}
	return vectors, nil
}

func (h *hostModels) setProfile(profile model.ConnectionProfile) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.profile = profile
}

func (h *hostModels) setProfileError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.profileErr = err
}

func (h *hostModels) countProfileCalls() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.profileCalls
}

func (h *hostModels) listEmbeddedCalls() []embedCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]embedCall(nil), h.embedded...)
}

func openAIProfile() model.ConnectionProfile {
	return model.ConnectionProfile{
		Model:      "text-embedding-3-small",
		Source:     "md.numen.connections.llm/openai https://api.openai.com/v1",
		Pooling:    "cls",
		Dimensions: 1536,
		MaxTokens:  8191,
	}
}

func connectionValues() map[string]string {
	return map[string]string{"source": "connection", "connections": `[{"connection":"c1","model":"text-embedding-3-small"}]`}
}

func newConnectionServer(host *hostModels) *Server {
	return newEmbeddingServerWith(func() (ports.Models, error) { return host, nil })
}

// The recipe of the vectors is known before any text is embedded: the type and the address of the
// connection are the source, and the width, the limit and the pooling are the profile's.
func TestAConnectionSourceReportsTheModelOfTheProfileBeforeAnyTextIsEmbedded(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	info := getModelInfoOf(t, srv)

	if info.Name != "text-embedding-3-small" || info.From != openAIProfile().Source || info.Dimensions != 1536 ||
		info.MaxTokens != 8191 || info.Pooling != "cls" {
		t.Errorf("model = %+v, want the profile of the connection", info)
	}
	if got := host.listEmbeddedCalls(); len(got) != 0 {
		t.Errorf("%d texts were embedded to know the model, want none", len(got))
	}
}

// Another address or another width is another recipe: the index is refitted and the old vectors are
// no longer the ones in use.
func TestAnotherAddressOrWidthChangesTheModelReported(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())
	before := getModelInfoOf(t, srv)

	moved := openAIProfile()
	moved.Source = "md.numen.connections.llm/openai-compatible http://127.0.0.1:11434/v1"
	host.setProfile(moved)
	srv.updateConfig(connectionValues())
	if after := getModelInfoOf(t, srv); after.From == before.From || after.Dimensions != before.Dimensions {
		t.Errorf("another address: from %q to %q, width %d to %d", before.From, after.From, before.Dimensions, after.Dimensions)
	}

	narrow := moved
	narrow.Dimensions = 768
	host.setProfile(narrow)
	srv.updateConfig(connectionValues())
	if after := getModelInfoOf(t, srv); after.Dimensions != 768 {
		t.Errorf("another width: %d, want 768", after.Dimensions)
	}
}

// The prefixes of the profile are part of what a vector is, so they are part of the handling hash.
func TestThePrefixesOfTheProfileChangeTheHandlingHash(t *testing.T) {
	unprefixed := &hostModels{profile: openAIProfile()}
	prefixed := openAIProfile()
	prefixed.QueryPrefix, prefixed.DocumentPrefix = "query: ", "passage: "
	other := &hostModels{profile: prefixed}

	first, second := newConnectionServer(unprefixed), newConnectionServer(other)
	first.updateConfig(connectionValues())
	second.updateConfig(connectionValues())

	if a, b := getModelInfoOf(t, first).Handling, getModelInfoOf(t, second).Handling; a == b || b == "" {
		t.Errorf("handling %q without prefixes and %q with them, want two different values", a, b)
	}
}

// The profile is asked for once for a configuration; a new configuration asks again, which is how an
// edited connection reaches the recipe.
func TestTheProfileIsReadOncePerConfiguration(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	for range 3 {
		getModelInfoOf(t, srv)
	}
	if got := host.countProfileCalls(); got != 1 {
		t.Fatalf("the profile was asked for %d times, want 1", got)
	}
	srv.updateConfig(connectionValues())
	getModelInfoOf(t, srv)
	if got := host.countProfileCalls(); got != 2 {
		t.Errorf("the profile was asked for %d times after a configuration, want 2", got)
	}
}

// A profile that could not be read is not remembered: the next read asks again.
func TestAProfileThatFailedIsAskedForAgain(t *testing.T) {
	host := &hostModels{profile: openAIProfile(), profileErr: ports.ConnectionError{Err: errors.New("OpenAI: unreachable")}}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())
	if _, err := srv.GetModelInfo(t.Context()); err == nil {
		t.Fatal("no error while the connection is unreachable")
	}

	host.setProfileError(nil)

	if info := getModelInfoOf(t, srv); info.Dimensions != 1536 {
		t.Errorf("width %d after the connection answered, want 1536", info.Dimensions)
	}
}

// Documents are embedded in the background and queries as a person's action, each with the prefix
// the profile names for its role.
func TestTextsAreEmbeddedWithThePrefixAndTheRoleOfTheirKind(t *testing.T) {
	profile := openAIProfile()
	profile.QueryPrefix, profile.DocumentPrefix = "query: ", "passage: "
	host := &hostModels{profile: profile}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.EmbedTexts(t.Context(), model.RoleQuery, []string{"c"}); err != nil {
		t.Fatal(err)
	}

	got := host.listEmbeddedCalls()
	if len(got) != 2 {
		t.Fatalf("%d requests, want 2", len(got))
	}
	if strings.Join(got[0].texts, "|") != "passage: a|passage: b" || got[0].role != model.RoleDocument {
		t.Errorf("documents: %+v", got[0])
	}
	if strings.Join(got[1].texts, "|") != "query: c" || got[1].role != model.RoleQuery {
		t.Errorf("queries: %+v", got[1])
	}
}

func TestTheVectorsOfTheConnectionAreReturnedInTheOrderOfTheTexts(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	vectors, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 3 || len(vectors[0]) != 1536 || vectors[2][0] != 3 {
		t.Errorf("vectors = %d of width %d", len(vectors), len(vectors[0]))
	}
}

// A vector of another width than the profile says would corrupt the index fitted to the profile.
func TestVectorsOfAnotherWidthThanTheProfileAreRefused(t *testing.T) {
	host := &hostModels{profile: openAIProfile(), width: 8}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	_, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"})

	if err == nil || !strings.Contains(err.Error(), "8") || !strings.Contains(err.Error(), "1536") {
		t.Fatalf("err = %v, want one that names both widths", err)
	}
}

// A connection that rejects the key or asks for payment stops the indexing with a text that names the
// connection and the reason.
func TestFailuresOfTheConnectionAreFailedPreconditionsThatNameTheReason(t *testing.T) {
	failure := ports.ConnectionError{Err: errors.New("OpenRouter: payment required")}
	host := &hostModels{profile: openAIProfile(), embedErr: failure}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	_, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"})

	if !isPrecondition(err) || !strings.Contains(err.Error(), "OpenRouter: payment required") {
		t.Errorf("err = %v, want a failed precondition naming the reason", err)
	}
	host.setProfileError(failure)
	srv.updateConfig(connectionValues())
	if _, err := srv.GetModelInfo(t.Context()); !isPrecondition(err) || !strings.Contains(err.Error(), "payment required") {
		t.Errorf("model info err = %v, want a failed precondition naming the reason", err)
	}
}

func TestACanceledCallKeepsItsContextError(t *testing.T) {
	host := &hostModels{profile: openAIProfile(), embedErr: context.Canceled}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	_, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"})

	if !errors.Is(err, context.Canceled) || isPrecondition(err) {
		t.Errorf("err = %v, want the cancellation", err)
	}
}

// A fresh install has no source, and stays on the local model without a call to the host.
func TestAFreshInstallStaysOnTheLocalModel(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	srv.updateConfig(map[string]string{})

	info := getModelInfoOf(t, srv)

	if info.Name != model.DefaultModel || info.Dimensions != 384 {
		t.Errorf("model = %s at %d, want the default local model", info.Name, info.Dimensions)
	}
	if host.countProfileCalls() != 0 {
		t.Error("the host was asked for a profile while the source is the local model")
	}
}

// The local model keeps its own settings and the change of source goes both ways.
func TestTheSourceSwitchesBetweenTheLocalModelAndTheConnection(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)

	srv.updateConfig(connectionValues())
	if info := getModelInfoOf(t, srv); info.Name != "text-embedding-3-small" {
		t.Fatalf("connection: %s", info.Name)
	}
	srv.updateConfig(map[string]string{"source": "local", "model": "Xenova/bge-m3"})
	if info := getModelInfoOf(t, srv); info.Name != "Xenova/bge-m3" || info.Dimensions != 1024 || info.From == openAIProfile().Source {
		t.Errorf("local: %+v", info)
	}
	srv.updateConfig(connectionValues())
	if info := getModelInfoOf(t, srv); info.Name != "text-embedding-3-small" {
		t.Errorf("connection again: %s", info.Name)
	}
}

// The settings provider and baseUrl of earlier versions are not read: a stored value changes nothing.
func TestTheRemovedProviderSettingIsIgnored(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	srv.updateConfig(map[string]string{"provider": "openai", "baseUrl": "https://example.org/v1"})

	info := getModelInfoOf(t, srv)

	if info.Name != model.DefaultModel {
		t.Errorf("model = %s, want the local default", info.Name)
	}
	if host.countProfileCalls() != 0 {
		t.Error("a stored provider sent the plugin to the host")
	}
}

func TestTheMockModelServesTheLocalSourceOnly(t *testing.T) {
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)

	srv.updateConfig(map[string]string{"source": "local"})
	if info := getModelInfoOf(t, srv); info.Name != "mock-embedding" {
		t.Fatalf("local: %s", info.Name)
	}
	srv.updateConfig(connectionValues())
	if info := getModelInfoOf(t, srv); info.Name != "text-embedding-3-small" {
		t.Errorf("connection: %s", info.Name)
	}
}

func TestAHostThatCannotBeReachedIsReportedWhenTheModelIsRead(t *testing.T) {
	srv := newEmbeddingServerWith(func() (ports.Models, error) { return nil, errors.New("no host") })
	srv.updateConfig(connectionValues())

	_, err := srv.GetModelInfo(t.Context())

	if err == nil || !strings.Contains(err.Error(), "no host") || !isPrecondition(err) {
		t.Errorf("err = %v", err)
	}
}

// The smallest piece and the longest chunk apply to the connection as they do to the local model; a
// chunk is limited to the token limit of the profile.
func TestTheChunkSettingsApplyToTheConnection(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	values := map[string]string{"minChunkLength": "120", "chunkTokens": "20000"}
	maps.Copy(values, connectionValues())
	srv.updateConfig(values)

	info := getModelInfoOf(t, srv)

	if info.MinChunkLength != 120 || info.ChunkTokens != 8191 {
		t.Errorf("minimum %d and chunk %d, want 120 and 8191", info.MinChunkLength, info.ChunkTokens)
	}
}

// The model of the connection runs elsewhere: the local model is not loaded for it.
func TestTheConnectionSourceNeverOpensTheLocalModel(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(host)
	isOpened := false
	srv.open = func(context.Context, model.ModelInfo, model.LocalModel) (ports.Embedder, error) {
		isOpened = true
		return funcEmbedder{embed: nil}, nil
	}
	srv.updateConfig(connectionValues())

	if _, err := srv.EmbedTexts(t.Context(), model.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}

	if isOpened {
		t.Error("the local model was opened for a connection")
	}
}

// A read of the profile that outlasts the caller goes on: the next call takes its result and asks the
// host nothing more.
func TestAProfileThatArrivesAfterTheCallerStoppedWaitingIsUsedByTheNextCall(t *testing.T) {
	host := &hostModels{profile: openAIProfile(), hold: make(chan struct{})}
	srv := newConnectionServer(host)
	srv.updateConfig(connectionValues())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := srv.GetModelInfo(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want the cancellation of the caller", err)
	}
	close(host.hold)

	info := getModelInfoOf(t, srv)

	if info.Dimensions != 1536 || host.countProfileCalls() != 1 {
		t.Errorf("width %d after %d reads of the profile, want 1536 after 1", info.Dimensions, host.countProfileCalls())
	}
}
