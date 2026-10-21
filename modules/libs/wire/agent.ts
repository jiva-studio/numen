/**
 * The agent port, answered from the wire.
 *
 * Every window asks the same agent the same way, so what a step on the wire
 * comes to in the port is written down once. The window makes the client; this
 * says what its answers mean.
 */
import type { AskAgentResponse, Span } from '@numen/protocol'
import type { AgentPort } from '@numen/ui'

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

/** The agent port, over the client the window talks on. */
export const createAgentPort = (agent: AgentClient): AgentPort => ({
  async *ask(query: Parameters<AgentPort['ask']>[0]) {
    const { question: prompt, focus, conversation, signal, agentId, modelId, reasoning } = query
    for await (const step of agent.askAgent(
      {
        prompt,
        focus,
        conversation,
        ...(agentId ? { agent: agentId } : {}),
        ...(modelId ? { model: modelId } : {}),
        ...(reasoning ? { reasoning } : {}),
      },
      { signal },
    )) {
      switch (step.step.case) {
        case 'text':
          yield { kind: 'said', text: step.step.value }
          break
        case 'toolCall': {
          const toolCall = step.step.value
          yield {
            kind: 'toolCall',
            tool: toolCall.tool,
            subject: toolCall.about,
            written: toolCall.length,
            ...(toolCall.path
              ? { place: { path: toolCall.path, span: normalizeSpan(toolCall.span) } }
              : {}),
          }
          break
        }
        case 'toolFinished':
          yield { kind: 'answered' }
          break
        case 'modelStarted':
          yield { kind: 'thinking' }
          break
        case 'stopped':
          yield { kind: 'stopped', error: step.step.value === STOPPED_DONE ? '' : step.step.value }
          break
      }
    }
  },

  async finish(conversation) {
    await agent.finishConversation({ conversation })
  },
})
