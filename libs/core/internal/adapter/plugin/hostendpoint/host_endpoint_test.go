package hostendpoint

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// callers is a host that knows the plugins it was told of and answers with the completer it was given.
type callers struct {
	mu        sync.Mutex
	plugins   map[string]string
	allowed   map[string]bool
	consented map[string]bool
	completer completionport.Completer
	checked   []string
	vault     pluginport.PluginVault
	texts     pluginport.PluginTexts
	work      pluginport.PluginWork
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
		return pluginport.ErrPermissionDenied
	}
	return nil
}

func (c *callers) CheckCompletionConsent(pluginID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checked = append(c.checked, pluginID+" consent")
	if !c.consented[pluginID] {
		return completionport.ErrNoConsent
	}
	return nil
}

func (c *callers) GetCompleter() completionport.Completer { return c.completer }

func (c *callers) GetPluginVault() pluginport.PluginVault { return c.vault }

func (c *callers) GetPluginTexts() pluginport.PluginTexts        { return c.texts }
func (c *callers) GetDependencies() pluginport.DependencyCatalog { return nil }
func (c *callers) GetPluginWork() pluginport.PluginWork          { return c.work }
func (c *callers) GetPluginTools() PluginTools                   { return c.tools }

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

// callerCompleter answers with the id of the plugin it was told asks.
type callerCompleter struct{}

func (callerCompleter) Complete(_ context.Context, pluginID string, _ completionport.Question) (string, error) {
	return pluginID, nil
}

func newCallers(completer completionport.Completer) *callers {
	return &callers{
		plugins:   map[string]string{"token-a": "md.numen.a", "token-b": "md.numen.b", "token-c": "md.numen.c"},
		allowed:   map[string]bool{"md.numen.a": true, "md.numen.c": true},
		consented: map[string]bool{"md.numen.a": true},
		completer: completer,
	}
}

// newHostClient serves the endpoint's handler and returns a client calling it with the token.
func newHostClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.HostServiceClient {
	t.Helper()
	endpoint := &Endpoint{callers: fakeCallers}
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

// A plugin holding the permission is answered with the text, and the completer
// is told the question as the plugin put it.
func TestAPluginWithThePermissionIsAnswered(t *testing.T) {
	var requestVal completionport.Question
	fakeCallers := newCallers(plugintest.CompleterFunc(func(_ context.Context, question completionport.Question) (string, error) {
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
	if len(fakeCallers.checked) != 2 || fakeCallers.checked[0] != "md.numen.a ai:complete" || fakeCallers.checked[1] != "md.numen.a consent" {
		t.Errorf("checked %v, want md.numen.a ai:complete, then its consent", fakeCallers.checked)
	}
}

// A plugin without the ai:complete permission fails, and nothing is asked of a model.
func TestAPluginWithoutThePermissionIsRefused(t *testing.T) {
	request := false
	fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completionport.Question) (string, error) {
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
			fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completionport.Question) (string, error) {
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
	endpoint := &Endpoint{callers: fakeCallers}
	for _, header := range []string{"", "token-a", "Basic token-a", "bearer token-a"} {
		if _, err := endpoint.authorize(header, plugindir.PermissionAIComplete); readCode(t, err) != connect.CodeUnauthenticated {
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
		{"no route", fmt.Errorf("%w: the program %q is not installed", completionport.ErrNoRoute, "p"), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, `the program "p" is not installed`},
		{"unreachable", fmt.Errorf("agent: %w", completionport.ErrUnavailable), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, "agent: unavailable"},
		{"stopped", fmt.Errorf("%w: the agent stopped: bad model", completionport.ErrUnavailable), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, "the agent stopped: bad model"},
	} {
		t.Run(one.name, func(t *testing.T) {
			fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completionport.Question) (string, error) { return "", one.err }))
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

// A host with nothing to answer completions refuses with NO_ROUTE.
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
	fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completionport.Question) (string, error) {
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
	fakeCallers := newCallers(plugintest.CompleterFunc(func(ctx context.Context, _ completionport.Question) (string, error) {
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
		t.Error("a cancelled call succeeded")
	}
}

// The endpoint listens on the loopback interface and answers for a token.
func TestTheHostEndpointListensOnLoopbackAndServes(t *testing.T) {
	fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completionport.Question) (string, error) { return "ok", nil }))
	endpoint, err := Open(t.Context(), Address, fakeCallers)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = endpoint.Close(context.WithoutCancel(t.Context())) })
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

// A plugin the person has not let ask a model is answered NO_CONSENT, and
// nothing is asked of a model.
func TestAPluginWithoutConsentIsAnsweredNoConsent(t *testing.T) {
	request := false
	fakeCallers := newCallers(plugintest.CompleterFunc(func(context.Context, completionport.Question) (string, error) {
		request = true
		return "", nil
	}))
	res, err := newHostClient(t, fakeCallers, "token-c").Complete(t.Context(), newHostQuestion())
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT || res.Msg.GetAnswer() != nil {
		t.Errorf("answered %v, want NO_CONSENT", res.Msg)
	}
	if request {
		t.Error("a model was asked for a plugin without consent")
	}
}

// A plugin without the permission fails before its consent is looked at.
func TestThePermissionIsCheckedBeforeTheConsent(t *testing.T) {
	fakeCallers := newCallers(nil)
	if _, err := newHostClient(t, fakeCallers, "token-b").Complete(t.Context(), newHostQuestion()); err == nil {
		t.Fatal("a plugin without the permission was served")
	}
	if slices.Contains(fakeCallers.checked, "md.numen.b consent") {
		t.Errorf("checked %v", fakeCallers.checked)
	}
}

// The completer is told the plugin the token names, whatever the question says.
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
	target := newCallers(plugintest.CompleterFunc(func(context.Context, completionport.Question) (string, error) {
		return "", fmt.Errorf("%w: no message", completionport.ErrInvalidQuestion)
	}))
	_, err := newHostClient(t, target, "token-a").Complete(t.Context(), newHostQuestion())
	if got := readCode(t, err); got != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want invalid argument", got)
	}
}
