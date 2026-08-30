/**
 * Asking about the card in front of a person.
 *
 * Nothing here draws: it is the client, and the words a question is put in.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { AgentService } from '@numen/protocol'
import type { AgentPort } from '@numen/ui'

import type { Asked } from '../core'

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

/**
 * The card a question is about, written where the agent will read it.
 *
 * The deck is the file in front of the person and goes as the focus; the card
 * is a mark inside that file, and the tools address it by that mark.
 */
export const standing = (card: Asked): string =>
  `The card is \`${card.card}\` in \`${card.deck}\`, shown through its ${card.face} face. ` +
  `Read it with card_read before answering.`
