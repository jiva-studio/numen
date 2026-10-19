package plugin

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// CapabilityAgentProvider is what a plugin declares to answer completions.
const CapabilityAgentProvider = "ai.completion.provider"

var (
	errNoCompletionAnswer = errors.New("agent plugin sent no answer")
	errUnknownRefusal     = errors.New("agent plugin refused with an unknown code")
	errNotAnAgent         = errors.New("plugin does not provide " + CapabilityAgentProvider)
)

// CompletionPluginHost provides plugin resolution and activation for the agent
// plugins that answer completions.
type CompletionPluginHost interface {
	port.PluginResolver
	port.PluginActivator
}

// CompletionAgents opens the agent plugin a route names.
type CompletionAgents struct {
	host CompletionPluginHost
}

// NewCompletionAgents returns the agents the host's plugins provide.
func NewCompletionAgents(host CompletionPluginHost) *CompletionAgents {
	return &CompletionAgents{host: host}
}

// OpenCompletionAgent activates the agent plugin with the id given, or, when the
// id is empty, the provider with the lowest id.
func (a *CompletionAgents) OpenCompletionAgent(ctx context.Context, agent string) (port.CompletionAgent, error) {
	pluginID, err := a.getAgentID(agent)
	if err != nil {
		return nil, err
	}
	bridge, err := OpenCompletionBridge(ctx, a.host, pluginID)
	if err != nil {
		return nil, readCompletionError(ctx, err)
	}
	return bridge, nil
}

func (a *CompletionAgents) getAgentID(agent string) (string, error) {
	if agent == "" {
		found, ok := a.host.FindPluginByCapability(CapabilityAgentProvider)
		if !ok {
			return "", fmt.Errorf("%w: no plugin provides %s", port.ErrNoRoute, CapabilityAgentProvider)
		}
		return found.Manifest.ID, nil
	}
	found, ok := a.host.GetPlugin(agent)
	if !ok {
		return "", fmt.Errorf("%w: no enabled plugin %q", port.ErrNoRoute, agent)
	}
	if !found.Manifest.HasCapability(CapabilityAgentProvider) {
		return "", fmt.Errorf("%w: %q: %w", port.ErrNoRoute, agent, errNotAnAgent)
	}
	return found.Manifest.ID, nil
}

// CompletionBridge connects a completion to an out-of-process agent plugin via Connect-RPC.
type CompletionBridge struct {
	service pluginService[numenv1connect.CompletionServiceClient]
	lease   *pluginLease
}

// OpenCompletionBridge activates or connects to the agent plugin with the id given.
func OpenCompletionBridge(ctx context.Context, host port.PluginActivator, pluginID string) (*CompletionBridge, error) {
	_, lease, err := openPlugin(ctx, host, pluginID, numenv1connect.NewCompletionServiceClient)
	if err != nil {
		return nil, err
	}
	service := pluginService[numenv1connect.CompletionServiceClient]{host: host, pluginID: pluginID, newClient: numenv1connect.NewCompletionServiceClient}
	return &CompletionBridge{service: service, lease: lease}, nil
}

// Complete asks the agent plugin to answer the question with the model named.
func (b *CompletionBridge) Complete(ctx context.Context, model string, question port.Question) (string, error) {
	client, call, err := b.service.open(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = call.Close() }()

	res, err := client.Complete(ctx, connect.NewRequest(&v1.CompleteRequest{
		Question: writeWireQuestion(question),
		Model:    model,
	}))
	if err != nil {
		return "", readCompletionError(ctx, err)
	}
	if err := readRefusal(res.Msg.GetError()); err != nil {
		return "", err
	}
	answer := res.Msg.GetAnswer()
	if answer == nil {
		return "", errNoCompletionAnswer
	}
	return answer.GetText(), nil
}

// Close releases the reference the bridge holds on the agent plugin, once.
func (b *CompletionBridge) Close() error {
	return b.lease.Close()
}

// readCompletionError sorts the failure of a call to an agent plugin: the end of
// the caller's context is the caller's own, an agent that cannot be reached is
// port.ErrUnavailable, and anything else the agent said is kept as it is.
func readCompletionError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
	}
	switch failure.Code() {
	case connect.CodeUnavailable, connect.CodeUnimplemented:
		return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
	default:
		return fmt.Errorf("agent complete: %w", err)
	}
}

// readRefusal maps the code of a refused call to its error. Unspecified is no refusal.
func readRefusal(code v1.CompletionErrorCode) error {
	switch code {
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED:
		return nil
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE:
		return port.ErrNoRoute
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE:
		return port.ErrUnavailable
	case v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT:
		return port.ErrNoConsent
	}
	return fmt.Errorf("%w: %d", errUnknownRefusal, code)
}

// writeRefusal maps an error of a completion to the code a refused call answers
// with. It reports false for an error that is no refusal.
func writeRefusal(err error) (v1.CompletionErrorCode, bool) {
	switch {
	case errors.Is(err, port.ErrNoConsent):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_CONSENT, true
	case errors.Is(err, port.ErrNoRoute):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_NO_ROUTE, true
	case errors.Is(err, port.ErrUnavailable):
		return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE, true
	}
	return v1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNSPECIFIED, false
}

func writeWireQuestion(question port.Question) *v1.CompletionQuestion {
	messages := make([]*v1.CompletionMessage, len(question.Messages))
	for i, message := range question.Messages {
		messages[i] = &v1.CompletionMessage{Role: message.Role, Text: message.Text}
	}
	return &v1.CompletionQuestion{
		Messages: messages,
		Purpose:  question.Purpose,
		Hints: &v1.CompletionHints{
			QualityPriority: question.Hints.QualityPriority,
			SpeedPriority:   question.Hints.SpeedPriority,
			CostPriority:    question.Hints.CostPriority,
			MaxTokens:       int32(question.Hints.MaxTokens),
			Temperature:     question.Hints.Temperature,
		},
	}
}

func readWireQuestion(wire *v1.CompletionQuestion) port.Question {
	messages := make([]port.Message, len(wire.GetMessages()))
	for i, message := range wire.GetMessages() {
		messages[i] = port.Message{Role: message.GetRole(), Text: message.GetText()}
	}
	hints := wire.GetHints()
	var temperature *float64
	if hints != nil && hints.Temperature != nil {
		value := hints.GetTemperature()
		temperature = &value
	}
	return port.Question{
		Messages: messages,
		Purpose:  wire.GetPurpose(),
		Hints: port.Hints{
			QualityPriority: hints.GetQualityPriority(),
			SpeedPriority:   hints.GetSpeedPriority(),
			CostPriority:    hints.GetCostPriority(),
			MaxTokens:       int(hints.GetMaxTokens()),
			Temperature:     temperature,
		},
	}
}

// ListModels returns the models the agent plugin answers completions with, in
// the order it lists them.
func (b *CompletionBridge) ListModels(ctx context.Context) ([]port.AgentModel, error) {
	client, call, err := b.service.open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = call.Close() }()

	res, err := client.ListCompletionModels(ctx, connect.NewRequest(&v1.ListCompletionModelsRequest{}))
	if err != nil {
		return nil, readCompletionError(ctx, err)
	}
	models := make([]port.AgentModel, 0, len(res.Msg.GetModels()))
	for _, one := range res.Msg.GetModels() {
		models = append(models, port.AgentModel{Name: one.GetName(), Title: one.GetTitle(), IsDefault: one.GetIsDefault()})
	}
	return models, nil
}
