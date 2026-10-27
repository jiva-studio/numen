/**
 * Asking about the card in front of a person.
 *
 * Nothing here draws: the port is the window's `AgentPort`, and this says what
 * the question is about.
 */
import type { AgentPort } from '@numen/plugin-sdk'
import type { ReviewCard } from '../../../entities/card'

/** The keys the card tools read the card in front of the person under. */
export const CARD_CONTEXT = {
  deck: 'md.numen.views.flashcards.studio/deck',
  card: 'md.numen.views.flashcards.studio/card',
  template: 'md.numen.views.flashcards.studio/template',
} as const

/**
 * The agent a conversation about one card is held with.
 *
 * Which card that is travels in the context of the question and never inside
 * it. A deck in a synced vault is named by whoever synced it, and a name in the
 * question is read as instruction; `card_current` gives the agent the same name
 * as a tool's answer, which is data.
 */
export const createCardAgentPort = (agent: AgentPort, card: ReviewCard): AgentPort => ({
  ask: (query) =>
    agent.ask({
      ...query,
      context: {
        ...query.context,
        [CARD_CONTEXT.deck]: card.deck,
        [CARD_CONTEXT.card]: card.cardId,
        [CARD_CONTEXT.template]: card.template,
      },
    }),
  finish: (conversation) => agent.finish(conversation),
})
