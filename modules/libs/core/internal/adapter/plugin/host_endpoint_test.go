package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// callers is a host that knows the plugins it was told of and answers with the completer it was given.
type callers struct {
	mu        sync.Mutex
	plugins   map[string]string
	allowed   map[string]bool
	completer port.Completer
	checked   []string
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
		return port.ErrPermissionDenied
	}
	return nil
}

func (c *callers) GetCompleter() port.Completer { return c.completer }

type completerFunc func(ctx context.Context, question port.Question) (string, error)

func (f completerFunc) Complete(ctx context.Context, question port.Question) (string, error) {
	return f(ctx, question)
}

func newCallers(completer port.Completer) *callers {
	return &callers{
		plugins:   map[string]string{"token-a": "md.numen.a", "token-b": "md.numen.b"},
		allowed:   map[string]bool{"md.numen.a": true},
		completer: completer,
	}
}

// newHostClient serves the endpoint's handler and returns a client calling it with the token.
func newHostClient(t *testing.T, held *callers, token string) numenv1connect.HostServiceClient {
	t.Helper()
	endpoint := &HostEndpoint{callers: held}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostServiceHandler(endpoint))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewHostServiceClient(server.Client(), server.URL, connect.WithInterceptors(authInterceptor(token)))
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
	var asked port.Question
	held := newCallers(completerFunc(func(_ context.Context, question port.Question) (string, error) {
		asked = question
		return "fixed", nil
	}))
	temperature := 0.25
	request := newHostQuestion()
	request.Msg.Question.Hints = &v1.CompletionHints{QualityPriority: 0.5, MaxTokens: 64, Temperature: &temperature}

	res, err := newHostClient(t, held, "token-a").Complete(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetAnswer().GetText() != "fixed" || res.Msg.GetError() != v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED {
		t.Errorf("answered %v", res.Msg)
	}
	if asked.Purpose != "proofreading" || len(asked.Messages) != 1 || asked.Messages[0].Text != "fix this" ||
		asked.Messages[0].Role != "user" || asked.Hints.QualityPriority != 0.5 || asked.Hints.MaxTokens != 64 ||
		asked.Hints.Temperature == nil || *asked.Hints.Temperature != temperature {
		t.Errorf("the completer was asked %+v", asked)
	}
	if len(held.checked) != 1 || held.checked[0] != "md.numen.a ai:complete" {
		t.Errorf("checked %v, want md.numen.a ai:complete", held.checked)
	}
}

// A plugin without the ai:complete permission is refused, and nothing is asked of a model.
func TestAPluginWithoutThePermissionIsRefused(t *testing.T) {
	asked := false
	held := newCallers(completerFunc(func(context.Context, port.Question) (string, error) {
		asked = true
		return "", nil
	}))

	_, err := newHostClient(t, held, "token-b").Complete(t.Context(), newHostQuestion())
	if got := readCode(t, err); got != connect.CodePermissionDenied {
		t.Errorf("code = %v, want permission denied", got)
	}
	if asked {
		t.Error("a model was asked for a plugin without the permission")
	}
}

// A caller with no token, an unknown token or a header that is no bearer is
// refused before the permission is looked at.
func TestACallerWithoutAKnownTokenIsRefused(t *testing.T) {
	for _, one := range []struct {
		name, token string
	}{
		{"none", ""},
		{"unknown", "token-x"},
	} {
		t.Run(one.name, func(t *testing.T) {
			held := newCallers(completerFunc(func(context.Context, port.Question) (string, error) {
				t.Error("a model was asked for a caller nobody knows")
				return "", nil
			}))
			_, err := newHostClient(t, held, one.token).Complete(t.Context(), newHostQuestion())
			if got := readCode(t, err); got != connect.CodeUnauthenticated {
				t.Errorf("code = %v, want unauthenticated", got)
			}
			if len(held.checked) != 0 {
				t.Errorf("checked %v for a caller nobody knows", held.checked)
			}
		})
	}
}

// A header that is no bearer token is not read as one.
func TestAHeaderThatIsNoBearerTokenIsRefused(t *testing.T) {
	held := newCallers(nil)
	endpoint := &HostEndpoint{callers: held}
	for _, header := range []string{"", "token-a", "Basic token-a", "bearer token-a"} {
		if err := endpoint.authorize(header, PermissionAIComplete); readCode(t, err) != connect.CodeUnauthenticated {
			t.Errorf("header %q was accepted", header)
		}
	}
}

// A call that was served and refused answers with the code, not with a status.
func TestARefusedCompletionAnswersWithItsCode(t *testing.T) {
	for _, one := range []struct {
		name string
		err  error
		want v1.CompletionErrorCode
	}{
		{"no route", fmt.Errorf("%w %q", port.ErrNoRoute, "p"), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE},
		{"unreachable", fmt.Errorf("agent: %w", port.ErrUnavailable), v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE},
	} {
		t.Run(one.name, func(t *testing.T) {
			held := newCallers(completerFunc(func(context.Context, port.Question) (string, error) { return "", one.err }))
			res, err := newHostClient(t, held, "token-a").Complete(t.Context(), newHostQuestion())
			if err != nil {
				t.Fatal(err)
			}
			if res.Msg.GetError() != one.want || res.Msg.GetAnswer() != nil {
				t.Errorf("answered %v, want code %v and no answer", res.Msg, one.want)
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

// A failure that is no refusal is a status.
func TestAFailureOfTheCompleterIsAStatus(t *testing.T) {
	held := newCallers(completerFunc(func(context.Context, port.Question) (string, error) {
		return "", errors.New("the model broke")
	}))
	_, err := newHostClient(t, held, "token-a").Complete(t.Context(), newHostQuestion())
	if got := readCode(t, err); got != connect.CodeInternal {
		t.Errorf("code = %v, want internal", got)
	}
}

// A plugin giving up ends the question the completer is answering.
func TestAPluginCancellingEndsTheQuestion(t *testing.T) {
	started := make(chan struct{})
	ended := make(chan error, 1)
	held := newCallers(completerFunc(func(ctx context.Context, _ port.Question) (string, error) {
		close(started)
		<-ctx.Done()
		ended <- ctx.Err()
		return "", ctx.Err()
	}))
	client := newHostClient(t, held, "token-a")

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
	held := newCallers(completerFunc(func(context.Context, port.Question) (string, error) { return "ok", nil }))
	endpoint, err := OpenHostEndpoint(t.Context(), hostEndpointAddress, held)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = endpoint.Close(context.WithoutCancel(t.Context())) })
	if endpoint.GetPort() <= 0 {
		t.Fatalf("port = %d", endpoint.GetPort())
	}

	client := numenv1connect.NewHostServiceClient(http.DefaultClient,
		fmt.Sprintf("http://127.0.0.1:%d", endpoint.GetPort()),
		connect.WithInterceptors(authInterceptor("token-a")))
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

// runningPlugin is a plugin the test host is told is running.
type runningPlugin struct {
	token string
	perms []string
}

func newRunningTestHost(t *testing.T, plugins map[string]runningPlugin) *Host {
	t.Helper()
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond) })
	for id, one := range plugins {
		sup := NewSupervisor()
		sup.SetPermissions(one.perms)
		host.supervisors[id] = sup
		host.endpoints[id] = Endpoint{Host: "127.0.0.1", Port: 1, Token: one.token}
	}
	return host
}

// The host names a running plugin by the token it was launched with.
func TestTheHostNamesAPluginByItsToken(t *testing.T) {
	host := newRunningTestHost(t, map[string]runningPlugin{
		"md.numen.a": {"token-a", []string{PermissionAIComplete}},
		"md.numen.b": {"token-b", nil},
		"md.numen.c": {"", nil},
	})

	for token, want := range map[string]string{"token-a": "md.numen.a", "token-b": "md.numen.b"} {
		if got, found := host.GetPluginIDByToken(token); !found || got != want {
			t.Errorf("token %q names %q, %v; want %q", token, got, found, want)
		}
	}
	for _, token := range []string{"", "token-", "token-ab", "other"} {
		if got, found := host.GetPluginIDByToken(token); found {
			t.Errorf("token %q names %q", token, got)
		}
	}
	if err := host.CheckPluginPermission("md.numen.b", PermissionAIComplete); !errors.Is(err, port.ErrPermissionDenied) {
		t.Errorf("a plugin without the permission was allowed: %v", err)
	}
	if err := host.CheckPluginPermission("md.numen.a", PermissionAIComplete); err != nil {
		t.Errorf("a plugin with the permission was refused: %v", err)
	}
}

// The host endpoint serves only once something answers completions, and
// stops when the host does.
func TestTheHostServesItsEndpointOnceItHasACompleter(t *testing.T) {
	host := NewHost("desktop.editor")
	if got, err := host.openHostEndpoint(t.Context()); err != nil || got != 0 {
		t.Fatalf("without a completer: port %d, %v", got, err)
	}

	host.SetCompleter(completerFunc(func(context.Context, port.Question) (string, error) { return "", nil }))
	first, err := host.openHostEndpoint(t.Context())
	if err != nil || first <= 0 {
		t.Fatalf("with a completer: port %d, %v", first, err)
	}
	if again, _ := host.openHostEndpoint(t.Context()); again != first {
		t.Errorf("a second call opened port %d, want %d", again, first)
	}

	if err := host.TeardownAll(t.Context(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got, _ := host.openHostEndpoint(t.Context()); got != 0 {
		t.Errorf("a closed host opened port %d", got)
	}
}

// The port reaches a plugin process in its environment, only while the
// endpoint serves, and the token it holds is the one the launch gave.
func TestThePluginEnvironmentCarriesTheHostPort(t *testing.T) {
	with := getPluginVars("md.numen.a", launchEnvironment{port: 5, hostPort: 7777, token: "t"})
	if !slices.Contains(with, "NUMEN_HOST_PORT=7777") || !slices.Contains(with, "NUMEN_PLUGIN_TOKEN=t") {
		t.Errorf("environment %v lacks the host port or the token", with)
	}
	without := getPluginVars("md.numen.a", launchEnvironment{port: 5, token: "t"})
	for _, entry := range without {
		if strings.HasPrefix(entry, "NUMEN_HOST_PORT=") {
			t.Errorf("environment %v names a host that is not serving", without)
		}
	}
}

// A plugin process launched while the host endpoint serves is told its port.
func TestALaunchedPluginIsToldTheHostPort(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond) })
	host.SetCompleter(completerFunc(func(context.Context, port.Question) (string, error) { return "", nil }))

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", 5, "t")
	if err != nil {
		t.Fatal(err)
	}
	served, _ := host.openHostEndpoint(t.Context())
	if want := fmt.Sprintf("NUMEN_HOST_PORT=%d", served); served == 0 || !slices.Contains(vars, want) {
		t.Errorf("environment %v lacks %s", vars, want)
	}
}

// An endpoint that cannot start is reported and does not stop a plugin from launching.
func TestAHostEndpointThatCannotStartDoesNotStopALaunch(t *testing.T) {
	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond) })
	host.hostAddress = "127.0.0.1:-1"
	host.SetCompleter(completerFunc(func(context.Context, port.Question) (string, error) { return "", nil }))
	var reported []error
	host.SetErrorHandler(func(err error) { reported = append(reported, err) })

	vars, err := host.getLaunchVars(t.Context(), "md.numen.a", 5, "t")
	if err != nil {
		t.Fatalf("the launch failed: %v", err)
	}
	for _, entry := range vars {
		if strings.HasPrefix(entry, "NUMEN_HOST_PORT=") {
			t.Errorf("environment %v names a host that is not serving", vars)
		}
	}
	if len(reported) != 1 || !strings.Contains(reported[0].Error(), "md.numen.a") {
		t.Errorf("reported %v", reported)
	}
}

// The host endpoint binds the loopback interface and refuses any address that
// would bind more.
func TestTheHostEndpointListensOnLoopbackOnly(t *testing.T) {
	held := newCallers(nil)
	for _, address := range []string{":0", "0.0.0.0:0", "[::]:0", "localhost:0", "192.168.1.5:0", "127.0.0.1", ""} {
		endpoint, err := OpenHostEndpoint(t.Context(), address, held)
		if err == nil {
			_ = endpoint.Close(t.Context())
			t.Errorf("address %q was bound", address)
		}
	}

	host := NewHost("desktop.editor")
	t.Cleanup(func() { _ = host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond) })
	if host.hostAddress != "127.0.0.1:0" {
		t.Errorf("a host listens on %q by default", host.hostAddress)
	}
	endpoint, err := OpenHostEndpoint(t.Context(), host.hostAddress, held)
	if err != nil {
		t.Fatal(err)
	}
	_ = endpoint.Close(t.Context())
}
