// Package health lets a plugin say whether it works. A plugin reports an item under a key of its own
// for each concern that does not work, such as "connection", and clears the key once it works again.
// The host lists the items of every plugin: an error item shows in the window's list of tasks and on
// the plugin's page, a warning item on the plugin's page only. The host clears the items of a plugin
// that is disabled, and keeps them while the plugin is stopped as idle.
//
// The host refuses an item past the limits: at most 8 keys, a key of lowercase letters, digits and
// hyphens of at most 32 characters, a text of at most 120 characters and a detail of at most 500. The
// keys configure, start and search are the host's.
package health

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

// Severity is how bad an item is.
type Severity int

const (
	// Warning is a plugin that works with less than it should.
	Warning Severity = iota + 1
	// Error is a plugin that does not work until a person acts.
	Error
)

// Item is one concern of the plugin that does not work.
//
//numen:api
type Item struct {
	// Key is the plugin's name for the concern, such as "connection".
	Key string
	// Severity is how bad it is.
	Severity Severity
	// Text is one sentence a person reads.
	Text string
	// Detail is more about it; empty has none.
	Detail string
	// Action is the fix a person runs; nil offers none.
	Action *Action
}

// Action is a command a person runs to fix what an item reports: an app: command, or a command of the
// plugin by its full id.
//
//numen:api
type Action struct {
	// Command is the id of the command.
	Command string
	// Values are the values of the arguments the command declares.
	Values map[string]string
	// Title is what the button reads; empty reads the title of the command.
	Title string
}

// OpenSettings is the action that opens the settings of the plugin with the setting of the key marked.
// An empty key opens the settings of the plugin.
func OpenSettings(setting string) *Action {
	values := map[string]string{}
	if setting != "" {
		values["setting"] = setting
	}
	return &Action{Command: "app:settings.open", Values: values}
}

// Client reports the items of the plugin to the host.
type Client struct {
	rpc numenv1connect.PluginHealthServiceClient
}

func NewClient(endpoint hostcall.Endpoint) *Client {
	return NewClientOver(hostcall.HTTPClient(), endpoint)
}

// NewClientOver returns the client of the host at the endpoint, sending its calls with httpClient.
func NewClientOver(httpClient connect.HTTPClient, endpoint hostcall.Endpoint) *Client {
	return &Client{rpc: numenv1connect.NewPluginHealthServiceClient(httpClient, endpoint.BaseURL, endpoint.Options()...)}
}

// NewClientFromEnv returns the client of the host the process was started for. It returns
// hostcall.ErrNoHost where the environment names none.
func NewClientFromEnv() (*Client, error) {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(endpoint), nil
}

// NewClientFromVars returns the client of the host the variables a linked plugin is given name. It
// returns hostcall.ErrNoHost where they name none.
func NewClientFromVars(vars map[string]string) (*Client, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	return NewClient(endpoint), nil
}

// Report puts the item in the host's list, or replaces the one under its key. An item the host refuses
// answers connect.CodeInvalidArgument.
func (c *Client) Report(ctx context.Context, item Item) error {
	_, err := c.rpc.ReportHealth(ctx, connect.NewRequest(&v1.ReportHealthRequest{Item: writeItem(item)}))
	return err
}

// Clear takes the item under the key out of the host's list. A key without an item is already cleared.
func (c *Client) Clear(ctx context.Context, key string) error {
	_, err := c.rpc.ClearHealth(ctx, connect.NewRequest(&v1.ClearHealthRequest{Key: key}))
	return err
}

func writeItem(item Item) *v1.HealthItem {
	out := &v1.HealthItem{Key: item.Key, Severity: writeSeverity(item.Severity), Text: item.Text, Detail: item.Detail}
	if item.Action != nil {
		out.Action = &v1.HealthAction{Command: item.Action.Command, Values: item.Action.Values, Title: item.Action.Title}
	}
	return out
}

func writeSeverity(severity Severity) v1.HealthSeverity {
	switch severity {
	case Warning:
		return v1.HealthSeverity_HEALTH_SEVERITY_WARNING
	case Error:
		return v1.HealthSeverity_HEALTH_SEVERITY_ERROR
	}
	return v1.HealthSeverity_HEALTH_SEVERITY_UNSPECIFIED
}
