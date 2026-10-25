package plugintool_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/plugintool"

	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

type offered struct {
	called string
}

func (o *offered) ListTools(context.Context) []pluginport.PluginTool {
	return []pluginport.PluginTool{{PluginID: "md.numen.sample", Name: "sample_read"}}
}

func (o *offered) CallTool(_ context.Context, pluginID, name string, arguments []byte, _ map[string]string) (pluginport.PluginToolResult, error) {
	o.called = pluginID + " " + name + " " + string(arguments)
	return pluginport.PluginToolResult{Text: "done"}, nil
}

func TestTheToolsThePluginsOfferAreListed(t *testing.T) {
	tools := plugintool.NewList(&offered{}).Execute(t.Context())

	if len(tools) != 1 || tools[0].Name != "sample_read" {
		t.Errorf("the tools were %+v", tools)
	}
}

func TestNoPluginsOfferNoTools(t *testing.T) {
	if tools := plugintool.NewList(nil).Execute(t.Context()); len(tools) != 0 {
		t.Errorf("the tools were %+v", tools)
	}
}

func TestACallIsHandedToThePluginThatListedTheTool(t *testing.T) {
	plugins := &offered{}

	result, err := plugintool.NewCall(plugins).Execute(t.Context(), "md.numen.sample", "sample_read", []byte(`{}`), nil)

	if err != nil || result.Text != "done" || plugins.called != "md.numen.sample sample_read {}" {
		t.Errorf("got %+v, %v after calling %q", result, err, plugins.called)
	}
}

func TestACallWithNoPluginsToAskIsUnavailable(t *testing.T) {
	_, err := plugintool.NewCall(nil).Execute(t.Context(), "md.numen.sample", "sample_read", nil, nil)

	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("the call was answered with %v", err)
	}
}
