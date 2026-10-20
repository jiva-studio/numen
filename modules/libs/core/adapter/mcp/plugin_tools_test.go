package mcp_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/plugintool"
)

// offeredTools is the tools some plugins offer, and the calls an agent made of them.
type offeredTools struct {
	tools []plugintool.Tool
	calls []offeredCall
	reply plugintool.Result
	err   error
}

type offeredCall struct {
	pluginID, name string
	arguments      string
}

func (o *offeredTools) ListTools(context.Context) []plugintool.Tool { return o.tools }

func (o *offeredTools) CallTool(
	_ context.Context, pluginID, name string, arguments []byte,
) (plugintool.Result, error) {
	o.calls = append(o.calls, offeredCall{pluginID, name, string(arguments)})
	return o.reply, o.err
}

const sampleSchema = `{"type":"object","properties":{"deckPath":{"type":"string"}},"required":["deckPath"]}`

func sampleTools() []plugintool.Tool {
	return []plugintool.Tool{
		{PluginID: "md.numen.sample", Name: "sample_read", Description: "Reads a sample.", InputSchema: []byte(sampleSchema), Effect: plugintool.EffectReads, Servers: []plugintool.Server{plugintool.ServerFull, plugintool.ServerReading}},
		{PluginID: "md.numen.sample", Name: "sample_write", Description: "Writes a sample.", InputSchema: []byte(sampleSchema), Effect: plugintool.EffectWrites},
		{PluginID: "md.numen.sample", Name: "sample_review", Description: "Reviews a sample.", InputSchema: []byte(sampleSchema), Effect: plugintool.EffectReads, Servers: []plugintool.Server{plugintool.ServerReview}},
		{PluginID: "md.numen.sample", Name: "sample_drop", Description: "Drops a sample.", InputSchema: []byte(sampleSchema), Effect: plugintool.EffectWrites, Step: plugintool.StepRemove},
	}
}

func servedWith(t *testing.T, plugins port.PluginTools, build func(mcp.Core) *sdk.Server) []string {
	t.Helper()
	_, core := newCore(t)
	core.Plugins = plugintool.NewTools(plugins)
	return getServedTools(t, sessionOf(t, build(core)))
}

func TestThePluginsToolsAreServedOnTheServersTheyNameAndNoOthers(t *testing.T) {
	plugins := &offeredTools{tools: sampleTools()}

	full := servedWith(t, plugins, mcp.New)
	reading := servedWith(t, plugins, mcp.NewReading)
	review := servedWith(t, plugins, mcp.NewReviewServer)

	for _, served := range []struct {
		name string
		got  []string
		want []string
		not  []string
	}{
		{"the full server", full, []string{"sample_read", "sample_write", "sample_drop"}, []string{"sample_review"}},
		{"the reading server", reading, []string{"sample_read"}, []string{"sample_write", "sample_drop", "sample_review"}},
		{"the review server", review, []string{"sample_review"}, []string{"sample_read", "sample_write"}},
	} {
		for _, name := range served.want {
			if !slices.Contains(served.got, name) {
				t.Errorf("%s does not serve %s", served.name, name)
			}
		}
		for _, name := range served.not {
			if slices.Contains(served.got, name) {
				t.Errorf("%s serves %s", served.name, name)
			}
		}
	}
}

func TestNoPluginsLeavesTheToolsOfTheCoreAlone(t *testing.T) {
	_, core := newCore(t)
	without := getServedTools(t, sessionOf(t, mcp.New(core)))

	empty := &offeredTools{}
	core.Plugins = plugintool.NewTools(empty)

	if with := getServedTools(t, sessionOf(t, mcp.New(core))); !slices.Equal(with, without) {
		t.Errorf("an empty list of plugin tools changed what is served:\n%v\n%v", with, without)
	}
}

func TestAToolTheCoreServesIsNotReplacedByAPluginsOfTheSameName(t *testing.T) {
	plugins := &offeredTools{tools: []plugintool.Tool{
		{PluginID: "md.numen.sample", Name: "note_read", Description: "Pretends.", InputSchema: []byte(sampleSchema), Effect: plugintool.EffectReads},
	}}
	_, core := newCore(t)
	core.Plugins = plugintool.NewTools(plugins)
	session := sessionOf(t, mcp.New(core))

	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "note_read", Arguments: map[string]any{"paths": []string{"Entropy.md"}}})
	if err != nil {
		t.Fatal(err)
	}

	if len(plugins.calls) != 0 || res.IsError {
		t.Errorf("the plugin answered a tool the core serves: %v, %v", plugins.calls, text(res))
	}
}

func TestACallIsHandedToThePluginThatListedTheToolWithItsArgumentsKeyedAsListed(t *testing.T) {
	plugins := &offeredTools{tools: sampleTools(), reply: plugintool.Result{Text: "done", Structured: []byte(`{"ok":true}`)}}
	_, core := newCore(t)
	core.Plugins = plugintool.NewTools(plugins)
	session := sessionOf(t, mcp.New(core))

	res, err := session.CallTool(t.Context(), &sdk.CallToolParams{Name: "sample_read", Arguments: map[string]any{"deck_path": "Animals.md"}})
	if err != nil {
		t.Fatal(err)
	}

	if res.IsError || text(res) != "done" {
		t.Errorf("the answer was %q (refused: %v)", text(res), res.IsError)
	}
	structured, err := json.Marshal(res.StructuredContent)
	if err != nil || string(structured) != `{"ok":true}` {
		t.Errorf("the structured answer was %s (%v)", structured, err)
	}
	want := []offeredCall{{"md.numen.sample", "sample_read", `{"deckPath":"Animals.md"}`}}
	if !slices.Equal(plugins.calls, want) {
		t.Errorf("the plugin was called %v, want %v", plugins.calls, want)
	}
}

func TestAToolThatRefusedOrFailedAnswersAsARefusal(t *testing.T) {
	refused := &offeredTools{tools: sampleTools(), reply: plugintool.Result{Text: "no such deck", IsRefusal: true}}
	failed := &offeredTools{tools: sampleTools(), err: port.ErrUnavailable}

	for name, plugins := range map[string]*offeredTools{"refused": refused, "failed": failed} {
		_, core := newCore(t)
		core.Plugins = plugintool.NewTools(plugins)
		session := sessionOf(t, mcp.New(core))

		if said := getRefusal(t, session, "sample_read", map[string]any{"deckPath": "x.md"}); said == "" {
			t.Errorf("a tool that %s said nothing", name)
		}
	}
}

func TestACallOfAPluginToolIsShownAsWhatTheToolSaysItDoes(t *testing.T) {
	_, core := newCore(t)
	plugins := &offeredTools{tools: sampleTools()}
	core.Plugins = plugintool.NewTools(plugins)

	words, err := mcp.Vocabulary(t.Context(), core)
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]domain.StepKind{
		"sample_read":  domain.StepRead,
		"sample_write": domain.StepEdit,
		"sample_drop":  domain.StepRemove,
	} {
		if got := words[name].Kind; got != want {
			t.Errorf("%s is shown as %v, want %v", name, got, want)
		}
	}
	if words["sample_read"].About != "deckPath" {
		t.Errorf("a call of sample_read is about %q", words["sample_read"].About)
	}
}
