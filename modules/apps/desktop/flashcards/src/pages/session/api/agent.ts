/**
 * Asking about the card in front of a person.
 *
 * Nothing here draws: the port is `AgentPort`, `@numen/wire` says what the
 * steps mean, and this says what the question is about.
 */
import { createClient } from '@connectrpc/connect'
import { AgentService } from '@numen/protocol'
import { agentPort, transport } from '@numen/wire'
import type { CardFace } from '@/entities/card'

const service = createClient(AgentService, transport)

/**
 * The agent a conversation about one card is held with.
 *
 * Which card that is travels beside the question and never inside it. A deck in
 * a synced vault is named by whoever synced it, and a name in the question is
 * read as instruction; `card_showing` gives the agent the same name as a tool's
 * answer, which is data.
 */
export const core = (card: CardFace) =>
  agentPort({
    askAgent: (request, options) =>
      service.askAgent({ ...request, mark: card.mark, face: card.face }, options),
    finishConversation: (request) => service.finishConversation(request),
  })
