package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/plugintest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// callers is a host that resolves the plugins registered with it and answers
// with the completer it was given.
type callers struct {
	mu        sync.Mutex
	plugins   map[string]string
	allowed   map[string]bool
	completer Completer
	checked   []string
	vault     *vaultapi.Access
	texts     TextVersions
	search    *SearchAccess
	work      WorkQueue
	window    WindowControl
	list      vaultapi.List
	granted   []string
	confined  bool
	tools     PluginTools
	acquired  map[string]int
	releases  chan string
}

func (c *callers) GetPluginIDByToken(token string) (string, bool) {
	id, found := c.plugins[token]
	return id, found && token != ""
}

func (c *callers) CheckPluginPermission(pluginID, permission string) error {
	c.mu.Lock()
	c.checked = append(c.checked, pluginID+" "+permission)
	c.mu.Unlock()
	if !c.allowed[pluginID] {
		return plugin.ErrPermissionDenied
	}
	if c.confined && !slices.Contains(c.granted, permission) {
		return plugin.ErrPermissionDenied
	}
	return nil
}

func (c *callers) over(list vaultapi.List, granted ...string) Callers {
	c.list, c.granted, c.confined = list, granted, true
	return c.asCallers()
}

func (c *callers) asCallers() Callers {
	lent := NewServices()
	lent.SetCompleter(c.completer)
	lent.SetPluginVault(c.vault)
	lent.SetPluginTexts(c.texts)
	lent.SetPluginSearch(c.search)
	lent.SetPluginWork(c.work)
	lent.SetPluginWindow(c.window)
	lent.SetPluginVaultList(c.list)
	return Callers{
		Identity: c, Leases: c, Lent: lent,
		Dependencies: func() Dependencies { return nil }, Tools: func() PluginTools { return c.tools },
	}
}

func capability[T any](fake any) T {
	found, isFound := fake.(T)
	if !isFound {
		var none T
		return none
	}
	return found
}

func vaultOf(fake any) *vaultapi.Access {
	return &vaultapi.Access{
		Notes: capability[vaultapi.NoteLookup](fake), Links: capability[vaultapi.LinkLookup](fake),
		Files: capability[vaultapi.FileReader](fake), Writer: capability[vaultapi.FileWriter](fake),
		Arranger: capability[vaultapi.Arranger](fake), LinkEditor: capability[vaultapi.LinkEditor](fake),
		Connections: capability[vaultapi.Connections](fake), Related: capability[vaultapi.RelatedNotes](fake),
		Watch: capability[vaultapi.Watch](fake), Switch: capability[vaultapi.Switch](fake),
		Stores: capability[vaultapi.StoreOpener](fake), Cache: capability[vaultapi.CacheStore](fake),
	}
}

func searchOf(fake any) *SearchAccess {
	if fake == nil {
		return nil
	}
	return &SearchAccess{
		SearchIndex: capability[SearchIndex](fake), PassageSearch: capability[PassageSearch](fake),
		SearchFloors: capability[SearchFloors](fake),
	}
}

func (c *callers) Acquire(pluginID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.acquired == nil {
		c.acquired = map[string]int{}
	}
	c.acquired[pluginID]++
}

func (c *callers) Release(pluginID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.acquired[pluginID]--
	if c.releases != nil {
		c.releases <- pluginID
	}
}

func (c *callers) getAcquired(pluginID string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.acquired[pluginID]
}

// callerCompleter answers with the id of the calling plugin.
type callerCompleter struct{}

func (callerCompleter) Complete(_ context.Context, pluginID string, _ completion.Question) (string, error) {
	return pluginID, nil
}

func newCallers(completer Completer) *callers {
	return &callers{
		plugins:   map[string]string{"token-a": "md.numen.a", "token-b": "md.numen.b", "token-c": "md.numen.c"},
		allowed:   map[string]bool{"md.numen.a": true, "md.numen.c": true},
		completer: completer,
	}
}

func newHostClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.HostServiceClient {
	t.Helper()
	endpoint := &Endpoint{callers: fakeCallers.asCallers()}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostServiceHandler(endpoint))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func newHostQuestion() *connect.Request[v1.HostServiceCompleteRequest] {
	return connect.NewRequest(&v1.HostServiceCompleteRequest{Question: &v1.CompletionQuestion{
		Purpose:  "proofreading",
		Messages: []*v1.CompletionMessage{{Role: "user", Text: "fix this"}},
	}})
}

func readCode(t *testing.T, err error) connect.Code {
	t.Helper()
	var failure *connect.Error
	if !errors.As(err, &failure) {
		t.Fatalf("error %v is not a status", err)
	}
	return failure.Code()
}

// A plugin holding the permission receives the text, and the completer
// receives the question as the plugin sent it.
func TestHostEndpoint_ServesPluginWithPermission(t *testing.T) {
	var requestVal completion.Question
	fakeCallers := newCallers(plugintest.CompleterFunc(func(_ context.Context, question completion.Question) (string, error) {
		requestVal = question
		return "fixed", nil
	}))
	request := newHostQuestion()

	res, err := newHostClient(t, fakeCallers, "token-a").Complete(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetAnswer().GetText() != "fixed" || res.Msg.GetError() != v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED {
		t.Errorf("answered %v", res.Msg)
	}
	if requestVal.Purpose != "proofreading" || len(requestVal.Messages) != 1 || requestVal.Messages[0].Text != "fix this" ||
		requestVal.Messages[0].Role != "user" {
		t.Errorf("the completer was asked %+v", requestVal)
	}
	if len(fakeCallers.checked) != 1 || fakeCallers.checked[0] != "md.numen.a ai:complete" {
		t.Errorf("checked %v, want md.numen.a ai:complete", fakeCallers.checked)
	}
}

// A plugin without the ai:complete permission fails, and no model is called.
func TestAPluginWithoutThePermissionIsRefused(t *testing.T) {
	request := false
	fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completion.Question) (string, error) {
		request = true
		return "", nil
	}))

	_, err := newHostClient(t, fakeCallers, "token-b").Complete(t.Context(), newHostQuestion())
	if got := readCode(t, err); got != connect.CodePermissionDenied {
		t.Errorf("code = %v, want permission denied", got)
	}
	if request {
		t.Error("a model was asked for a plugin without the permission")
	}
}

// A caller with no token, an unknown token or a header that is no bearer is
func TestACallerWithoutAKnownTokenIsRefused(t *testing.T) {
	for _, one := range []struct {
		name, token string
	}{
		{"none", ""},
		{"unknown", "token-x"},
	} {
		t.Run(one.name, func(t *testing.T) {
			fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completion.Question) (string, error) {
				t.Error("a model was asked for a caller nobody knows")
				return "", nil
			}))
			_, err := newHostClient(t, fakeCallers, one.token).Complete(t.Context(), newHostQuestion())
			if got := readCode(t, err); got != connect.CodeUnauthenticated {
				t.Errorf("code = %v, want unauthenticated", got)
			}
			if len(fakeCallers.checked) != 0 {
				t.Errorf("checked %v for a caller nobody knows", fakeCallers.checked)
			}
		})
	}
}

// A header that is no bearer token is not read as one.
func TestAHeaderThatIsNoBearerTokenIsRefused(t *testing.T) {
	fakeCallers := newCallers(nil)
	endpoint := &Endpoint{callers: fakeCallers.asCallers()}
	for _, header := range []string{"", "token-a", "Basic token-a", "bearer token-a"} {
		if _, err := endpoint.Authorize(header, manifest.PermissionAIComplete); readCode(t, err) != connect.CodeUnauthenticated {
			t.Errorf("header %q was accepted", header)
		}
	}
}

func TestARefusedCompletionAnswersWithItsCode(t *testing.T) {
	for _, one := range []struct {
		name   string
		err    error
		want   v1.CompletionErrorCode
		detail string
	}{
		{"no route", fmt.Errorf("%w: the program %q is not installed", completion.ErrNoRoute, "p"), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, `the program "p" is not installed`},
		{"unreachable", fmt.Errorf("agent: %w", domain.ErrUnavailable), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, "agent: unavailable"},
		{"stopped", fmt.Errorf("%w: the agent stopped: bad model", domain.ErrUnavailable), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, "the agent stopped: bad model"},
	} {
		t.Run(one.name, func(t *testing.T) {
			fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completion.Question) (string, error) { return "", one.err }))
			res, err := newHostClient(t, fakeCallers, "token-a").Complete(t.Context(), newHostQuestion())
			if err != nil {
				t.Fatal(err)
			}
			if res.Msg.GetError() != one.want || res.Msg.GetAnswer() != nil {
				t.Errorf("answered %v, want code %v and no answer", res.Msg, one.want)
			}
			if res.Msg.GetDetail() != one.detail {
				t.Errorf("detail %q, want %q", res.Msg.GetDetail(), one.detail)
			}
		})
	}
}

// A host without a completion backend returns NO_ROUTE.
func TestAHostWithoutACompleterAnswersNoRoute(t *testing.T) {
	res, err := newHostClient(t, newCallers(nil), "token-a").Complete(t.Context(), newHostQuestion())
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE {
		t.Errorf("answered %v", res.Msg)
	}
}

func TestAFailureOfTheCompleterIsAStatus(t *testing.T) {
	fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completion.Question) (string, error) {
		return "", errors.New("the model broke")
	}))
	_, err := newHostClient(t, fakeCallers, "token-a").Complete(t.Context(), newHostQuestion())
	if got := readCode(t, err); got != connect.CodeInternal {
		t.Errorf("code = %v, want internal", got)
	}
}

// A plugin giving up ends the question the completer is answering.
func TestAPluginCancellingEndsTheQuestion(t *testing.T) {
	started := make(chan struct{})
	ended := make(chan error, 1)
	fakeCallers := newCallers(plugintest.CompleterFunc(func(ctx context.Context, _ completion.Question) (string, error) {
		close(started)
		<-ctx.Done()
		ended <- ctx.Err()
		return "", ctx.Err()
	}))
	client := newHostClient(t, fakeCallers, "token-a")

	ctx, cancel := context.WithCancel(t.Context())
	failed := make(chan error, 1)
	go func() {
		_, err := client.Complete(ctx, newHostQuestion())
		failed <- err
	}()
	<-started
	cancel()

	select {
	case err := <-ended:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the completer saw %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the question went on after the plugin gave up")
	}
	if err := <-failed; err == nil {
		t.Error("a canceled call succeeded")
	}
}

// The endpoint binds the loopback interface and responds for a token.
func TestHostEndpoint_BindsLoopbackAndServes(t *testing.T) {
	fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completion.Question) (string, error) { return "ok", nil }))
	endpoint, err := Open(t.Context(), Address, fakeCallers.asCallers())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := endpoint.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	if endpoint.GetPort() <= 0 {
		t.Fatalf("port = %d", endpoint.GetPort())
	}

	client := numenv1connect.NewHostServiceClient(http.DefaultClient,
		fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort()),
		connect.WithInterceptors(transport.AuthInterceptor("token-a")))
	res, err := client.Complete(t.Context(), newHostQuestion())
	if err != nil || res.Msg.GetAnswer().GetText() != "ok" {
		t.Fatalf("answered %v, %v", res, err)
	}

	if err := endpoint.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Complete(t.Context(), newHostQuestion()); err == nil {
		t.Error("a closed endpoint answered")
	}
}

// The completer receives the plugin identified by the token, regardless of the
// question's content.
func TestTheCallerIsTheTokensPlugin(t *testing.T) {
	request := newHostQuestion()
	request.Msg.Question.Purpose = "md.numen.c"
	res, err := newHostClient(t, newCallers(callerCompleter{}), "token-a").Complete(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Msg.GetAnswer().GetText(); got != "md.numen.a" {
		t.Errorf("the completer received the caller is %q, want md.numen.a", got)
	}
}

// A question nothing can answer is an invalid argument, not an error.
func TestAQuestionNothingCanAnswerIsAnInvalidArgument(t *testing.T) {
	target := newCallers(plugintest.CompleterFunc(func(context.Context, completion.Question) (string, error) {
		return "", fmt.Errorf("%w: no message", completion.ErrInvalidQuestion)
	}))
	_, err := newHostClient(t, target, "token-a").Complete(t.Context(), newHostQuestion())
	if got := readCode(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want invalid argument", got)
	}
}
