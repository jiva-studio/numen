/**
 * The agent port implemented over the wire.
 *
 * Every window queries the same agent the same way, so the meaning of each wire
 * step in the port is defined once. The window creates the client; this module
 * defines what its responses mean.
 */
import type { AgentPort, AgentQuery, AgentStep } from '../agent'

/** A run of text on the wire, by where it begins and where it ends. */
interface WireRange {
  readonly start: number
  readonly end: number
}

/** A tool the agent is using, as the wire carries it. */
interface WireToolCall {
  readonly tool: string
  readonly about: string
  readonly length: number
  readonly path?: string | undefined
  readonly range?: WireRange | undefined
}

/** One step of the work on the wire. */
interface WireAgentResponse {
  readonly step:
    | { readonly case: 'text'; readonly value: string }
    | { readonly case: 'toolCall'; readonly value: WireToolCall }
    | { readonly case: 'stopped'; readonly value: string }
    | { readonly case: 'toolFinished'; readonly value?: unknown }
    | { readonly case: 'modelStarted'; readonly value?: unknown }
    | { readonly case: 'editProgress'; readonly value?: unknown }
    | { readonly case: undefined; readonly value?: undefined }
}

/** The range a call named, and an empty one for a call naming no place inside the source. */
const normalizeRange = (range: WireRange | undefined) => ({
  start: range?.start ?? 0,
  end: range?.end ?? 0,
})

/** The reason a stopped step gives when the agent stopped because it was done. */
const STOPPED_DONE = 'done'

/** The subset of the agent service that the port uses. */
export interface AgentClient {
  askAgent(
    request: {
      prompt: string
      focus: string
      conversation: string
      agent?: string
      model?: string
      reasoning?: string
      context?: Record<string, string>
      vaultId?: string
    },
    options: { signal: AbortSignal },
  ): AsyncIterable<WireAgentResponse>
  finishConversation(request: { conversation: string }): Promise<unknown>
}

type AgentRequest = Parameters<AgentClient['askAgent']>[0]

/** The request built from a query. An empty agent, model, reasoning, context or vault is omitted. */
const createAgentRequest = (query: AgentQuery): AgentRequest => {
  const {
    question: prompt,
    focus,
    conversation,
    agentId,
    modelId,
    reasoning,
    context,
    vault,
  } = query
  return {
    prompt,
    focus,
    conversation,
    ...(agentId ? { agent: agentId } : {}),
    ...(modelId ? { model: modelId } : {}),
    ...(reasoning ? { reasoning } : {}),
    ...(context && Object.keys(context).length > 0 ? { context: { ...context } } : {}),
    ...(vault ? { vaultId: vault } : {}),
  }
}

/** The step of the port a step on the wire is, or undefined for one that carries none. */
const getAgentStep = (step: WireAgentResponse): AgentStep | undefined => {
  switch (step.step.case) {
    case 'text':
      return { kind: 'text', text: step.step.value }
    case 'toolCall': {
      const toolCall = step.step.value
      return {
        kind: 'toolCall',
        tool: toolCall.tool,
        subject: toolCall.about,
        written: toolCall.length,
        ...(toolCall.path
          ? { place: { path: toolCall.path, range: normalizeRange(toolCall.range) } }
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
      if (converted) {
        yield converted
        if (converted.kind === 'stopped') return
      }
    }
  },

  async finish(conversation) {
    await agent.finishConversation({ conversation })
  },
})
