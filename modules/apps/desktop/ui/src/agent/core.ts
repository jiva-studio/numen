/**
 * Asking an agent about the vault the window is showing.
 *
 * The window hands over what was written and reads what the agent does as it
 * does it. The port is `AgentPort`, and this is what answers it over the wire.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { AgentService } from '@numen/protocol'
import type { AgentPort } from '@numen/ui'

const agent = createClient(
  AgentService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

export const core: AgentPort = {
  async *ask(asked, focus, conversation, signal) {
    for await (const step of agent.ask({ asked, focus, conversation }, { signal })) {
      switch (step.step.case) {
        case 'said':
          yield { kind: 'said', text: step.step.value }
          break
        case 'doing': {
          const said = step.step.value
          yield {
            kind: 'doing',
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
}
