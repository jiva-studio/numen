/**
 * The agent port, answered from the wire.
 *
 * Every window asks the same agent the same way, so what a step on the wire
 * evaluates to in the port is written down once. The window makes the client; this
 * says what its answers mean.
 */
import type { AskAgentResponse, Span } from '@numen/protocol'
import type { AgentPort, AgentQuery, AgentStep } from '@numen/plugin-sdk'

/** The span a call named, and one naming no place inside the source. */
const normalizeSpan = (span: Span | undefined) => ({ from: span?.from ?? 0, to: span?.to ?? 0 })

/** The reason a stopped step gives when the agent stopped because it was done. */
const STOPPED_DONE = 'done'

/** As much of the agent service as the port asks of it. */
export interface AgentClient {
  askAgent(
    request: {
      prompt: string
      focus: string
      conversation: string
      agent?: string
      model?: string
      reasoning?: string
    },
    options: { signal: AbortSignal },
  ): AsyncIterable<AskAgentResponse>
  finishConversation(request: { conversation: string }): Promise<unknown>
}

type AgentRequest = Parameters<AgentClient['askAgent']>[0]

/** The request a query is asked with. An empty agent, model or reasoning is left out. */
const createAgentRequest = (query: AgentQuery): AgentRequest => {
  const { question: prompt, focus, conversation, agentId, modelId, reasoning } = query
  return {
    prompt,
    focus,
    conversation,
    ...(agentId ? { agent: agentId } : {}),
    ...(modelId ? { model: modelId } : {}),
    ...(reasoning ? { reasoning } : {}),
  }
}

/** The step of the port a step on the wire is, or undefined for one that carries none. */
const getAgentStep = (step: AskAgentResponse): AgentStep | undefined => {
  switch (step.step.case) {
    case 'text':
      return { kind: 'said', text: step.step.value }
    case 'toolCall': {
      const toolCall = step.step.value
      return {
        kind: 'toolCall',
        tool: toolCall.tool,
        subject: toolCall.about,
        written: toolCall.length,
        ...(toolCall.path
          ? { place: { path: toolCall.path, span: normalizeSpan(toolCall.span) } }
          : {}),
      }
    }
    case 'toolFinished':
      return { kind: 'answered' }
    case 'modelStarted':
      return { kind: 'thinking' }
    case 'stopped':
      return { kind: 'stopped', error: step.step.value === STOPPED_DONE ? '' : step.step.value }
    default:
      return undefined
  }
}

/** The agent port, over the client the window talks on. */
export const createAgentPort = (agent: AgentClient): AgentPort => ({
  async *ask(query: AgentQuery) {
    for await (const step of agent.askAgent(createAgentRequest(query), { signal: query.signal })) {
      const converted = getAgentStep(step)
      if (converted) yield converted
    }
  },

  async finish(conversation) {
    await agent.finishConversation({ conversation })
  },
})
