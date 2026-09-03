/**
 * The agent port, answered from the wire.
 *
 * Every window asks the same agent the same way, so what a step on the wire
 * comes to in the port is written down once. The window makes the client; this
 * says what its answers mean.
 */
import type { AskResponse } from '@numen/protocol'
import type { AgentPort } from '@numen/ui'

/** As much of the agent service as the port asks of it. */
export interface Asking {
  ask(
    request: { asked: string; focus: string; conversation: string },
    options: { signal: AbortSignal },
  ): AsyncIterable<AskResponse>
  finish(request: { conversation: string }): Promise<unknown>
}

/** The agent port, over the client the window talks on. */
export const agentPort = (agent: Asking): AgentPort => ({
  async *ask(asked, focus, conversation, signal) {
    for await (const step of agent.ask({ asked, focus, conversation }, { signal })) {
      switch (step.step.case) {
        case 'said':
          yield { kind: 'said', text: step.step.value }
          break
        case 'toolCall': {
          const said = step.step.value
          yield {
            kind: 'toolCall',
            tool: said.tool,
            about: said.about,
            written: said.written,
            ...(said.path
              ? { place: { path: said.path, start: said.start, length: said.length } }
              : {}),
          }
          break
        }
        case 'answered':
          yield { kind: 'answered' }
          break
        case 'thinking':
          yield { kind: 'thinking' }
          break
        case 'stopped':
          yield { kind: 'stopped', failed: step.step.value }
          break
      }
    }
  },

  async finish(conversation) {
    await agent.finish({ conversation })
  },
})
