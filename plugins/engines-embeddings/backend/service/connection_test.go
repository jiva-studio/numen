package service

import (
	"context"
	"errors"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/rpcerror"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
)

// hostModels simulates the model API of the host: it answers with a profile and embeds with the
// width of that profile, and it records what it was asked.
type hostModels struct {
	mu           sync.Mutex
	profile      connection.EmbeddingProfile
	profileErr   error
	embedErr     error
	width        int
	profileCalls int
	hold         chan struct{}
	embedded     []connection.EmbedRequest
}

func (h *hostModels) GetEmbeddingProfile(context.Context) (connection.EmbeddingProfile, error) {
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

func (h *hostModels) Embed(_ context.Context, request connection.EmbedRequest) (connection.EmbedResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.embedded = append(h.embedded, request)
	if h.embedErr != nil {
		return connection.EmbedResult{}, h.embedErr
	}
	width := h.width
	if width == 0 {
		width = h.profile.Dimensions
	}
	vectors := make([][]float32, len(request.Texts))
	for i := range vectors {
		vectors[i] = make([]float32, width)
		vectors[i][0] = float32(i + 1)
	}
	return connection.EmbedResult{Vectors: vectors}, nil
}

func (h *hostModels) setProfile(profile connection.EmbeddingProfile) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.profile = profile
}

func (h *hostModels) countProfileCalls() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.profileCalls
}

func (h *hostModels) listEmbedded() []connection.EmbedRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]connection.EmbedRequest(nil), h.embedded...)
}

func openAIProfile() connection.EmbeddingProfile {
	return connection.EmbeddingProfile{
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

func newConnectionServer(t *testing.T, host *hostModels) *embeddingServer {
	t.Helper()
	srv := newEmbeddingServer()
	srv.newModels = func() (modelClient, error) { return host, nil }
	return srv
}

func getModelInfoOf(t *testing.T, srv *embeddingServer) embedding.ModelInfo {
	t.Helper()
	info, err := srv.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return info
}

// The recipe of the vectors is known before any text is embedded: the type and the address of the
// connection are the source, and the width, the limit and the pooling are the profile's.
func TestAConnectionSourceReportsTheModelOfTheProfileBeforeAnyTextIsEmbedded(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)
	srv.updateConfig(connectionValues())

	info := getModelInfoOf(t, srv)

	if info.Name != "text-embedding-3-small" || info.From != openAIProfile().Source || info.Dimensions != 1536 ||
		info.MaxTokens != 8191 || info.Pooling != "cls" {
		t.Errorf("model = %+v, want the profile of the connection", info)
	}
	if got := host.listEmbedded(); len(got) != 0 {
		t.Errorf("%d texts were embedded to know the model, want none", len(got))
	}
}

// Another address or another width is another recipe: the index is refitted and the old vectors are
// no longer the ones in use.
func TestAnotherAddressOrWidthChangesTheModelReported(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)
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

	first, second := newConnectionServer(t, unprefixed), newConnectionServer(t, other)
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
	srv := newConnectionServer(t, host)
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
	host := &hostModels{profile: openAIProfile(), profileErr: &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "OpenAI", Reason: "unreachable"}}}}
	srv := newConnectionServer(t, host)
	srv.updateConfig(connectionValues())
	if _, err := srv.GetModelInfo(t.Context()); err == nil {
		t.Fatal("no error while the connection is unreachable")
	}

	host.mu.Lock()
	host.profileErr = nil
	host.mu.Unlock()
	if info := getModelInfoOf(t, srv); info.Dimensions != 1536 {
		t.Errorf("width %d after the connection answered, want 1536", info.Dimensions)
	}
}

// Documents are embedded in the background and queries as a person's action, each with the prefix
// the profile names for its role.
func TestTextsAreEmbeddedWithThePrefixAndThePriorityOfTheirRole(t *testing.T) {
	profile := openAIProfile()
	profile.QueryPrefix, profile.DocumentPrefix = "query: ", "passage: "
	host := &hostModels{profile: profile}
	srv := newConnectionServer(t, host)
	srv.updateConfig(connectionValues())

	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleQuery, []string{"c"}); err != nil {
		t.Fatal(err)
	}

	got := host.listEmbedded()
	if len(got) != 2 {
		t.Fatalf("%d requests, want 2", len(got))
	}
	if strings.Join(got[0].Texts, "|") != "passage: a|passage: b" || got[0].Priority != connection.PriorityBackground {
		t.Errorf("documents: %+v", got[0])
	}
	if strings.Join(got[1].Texts, "|") != "query: c" || got[1].Priority != connection.PriorityInteractive {
		t.Errorf("queries: %+v", got[1])
	}
}

func TestTheVectorsOfTheConnectionAreReturnedInTheOrderOfTheTexts(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)
	srv.updateConfig(connectionValues())

	vectors, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a", "b", "c"})
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
	srv := newConnectionServer(t, host)
	srv.updateConfig(connectionValues())

	_, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"})

	if err == nil || !strings.Contains(err.Error(), "8") || !strings.Contains(err.Error(), "1536") {
		t.Fatalf("err = %v, want one that names both widths", err)
	}
}

// A connection that rejects the key or asks for payment stops the indexing with a text that names the
// connection and the reason.
func TestFailuresOfTheConnectionNameTheConnectionAndTheReason(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"every row failed", &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "OpenRouter", Reason: "payment required"}}}, "OpenRouter: payment required"},
		{"a row rejected the request", &connection.RejectedError{ConnectionName: "Ollama", Reason: "key rejected"}, "Ollama"},
		{"no row", connection.ErrNoConnection, "no connection"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := &hostModels{profile: openAIProfile(), embedErr: tc.err}
			srv := newConnectionServer(t, host)
			srv.updateConfig(connectionValues())

			_, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"})

			if rpcerror.GetCode(err) != rpcerror.FailedPrecondition || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v (%v), want FailedPrecondition naming %q", err, rpcerror.GetCode(err), tc.want)
			}
			host.mu.Lock()
			host.profileErr = tc.err
			host.mu.Unlock()
			srv.updateConfig(connectionValues())
			if _, err := srv.GetModelInfo(t.Context()); rpcerror.GetCode(err) != rpcerror.FailedPrecondition || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("model info err = %v, want FailedPrecondition naming %q", err, tc.want)
			}
		})
	}
}

func TestACancelledCallKeepsItsContextError(t *testing.T) {
	host := &hostModels{profile: openAIProfile(), embedErr: context.Canceled}
	srv := newConnectionServer(t, host)
	srv.updateConfig(connectionValues())

	_, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the cancellation", err)
	}
}

// A fresh install has no source, and stays on the local model without a call to the host.
func TestAFreshInstallStaysOnTheLocalModel(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)
	srv.updateConfig(map[string]string{})

	info := getModelInfoOf(t, srv)

	if info.Name != engine.DefaultModel || info.Dimensions != 384 {
		t.Errorf("model = %s at %d, want the default local model", info.Name, info.Dimensions)
	}
	if host.countProfileCalls() != 0 {
		t.Error("the host was asked for a profile while the source is the local model")
	}
}

// The local model keeps its own settings and the change of source goes both ways.
func TestTheSourceSwitchesBetweenTheLocalModelAndTheConnection(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)

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

// The setting `provider` of earlier versions is not read: a stored value changes nothing.
func TestTheRemovedProviderSettingIsIgnored(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)
	srv.updateConfig(map[string]string{"provider": "openai", "baseUrl": "https://example.org/v1"})

	info := getModelInfoOf(t, srv)

	if info.Name != engine.DefaultModel {
		t.Errorf("model = %s, want the local default", info.Name)
	}
	if host.countProfileCalls() != 0 {
		t.Error("a stored provider sent the plugin to the host")
	}
}

// The environment names no address of a service.
func TestTheEnvironmentNamesNoServiceAddress(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_URL", "https://example.org/v1")
	t.Setenv(envMockML, "true")
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)
	srv.updateConfig(map[string]string{})

	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if len(host.listEmbedded()) != 0 {
		t.Error("texts left for the host while the source is the local model")
	}
}

// The mock model of the tests stands for the local model only.
func TestTheMockModelServesTheLocalSourceOnly(t *testing.T) {
	t.Setenv(envMockML, "true")
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)

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
	srv := newEmbeddingServer()
	srv.newModels = func() (modelClient, error) { return nil, errors.New("no host") }
	srv.updateConfig(connectionValues())

	_, err := srv.GetModelInfo(t.Context())

	if err == nil || !strings.Contains(err.Error(), "no host") {
		t.Errorf("err = %v", err)
	}
}

// The smallest piece and the longest chunk apply to the connection as they do to the local model; a
// chunk is limited to the token limit of the profile.
func TestTheChunkSettingsApplyToTheConnection(t *testing.T) {
	host := &hostModels{profile: openAIProfile()}
	srv := newConnectionServer(t, host)
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
	srv := newConnectionServer(t, host)
	isOpened := false
	srv.open = func(context.Context, engine.ModelInfo, engine.LocalModel) (textEmbedder, error) {
		isOpened = true
		return engine.MockEmbedder{}, nil
	}
	srv.updateConfig(connectionValues())

	if _, err := srv.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"a"}); err != nil {
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
	srv := newConnectionServer(t, host)
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
