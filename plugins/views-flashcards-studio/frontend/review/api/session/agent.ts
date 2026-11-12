/**
 * Questions about the card currently displayed.
 *
 * Nothing here renders: the port is the window's `AgentPort`, and this module
 * adds the card the question is about.
 */
import type { AgentPort } from '@numen/plugin-sdk'
import type { ReviewCard } from '../../lib/card/cards'

/** The keys under which the card tools read the card currently displayed. */
const CARD_CONTEXT = {
  deck: 'md.numen.views.flashcards.studio/deck',
  card: 'md.numen.views.flashcards.studio/card',
  template: 'md.numen.views.flashcards.studio/template',
} as const

/**
 * The agent for a conversation about one card of a vault.
 * The card identity goes in the context of the question, not in its text: a deck name in a synced
 * vault is set by the sync source, and in the question it would be read as an instruction, while
 * `card_current_get` returns it as a tool result, which is data. The window has no vault of its
 * own, so the question names the vault the card tools operate on.
 */
export const createCardAgentPort = (
  agent: AgentPort,
  card: ReviewCard,
  vault: string,
): AgentPort => ({
  ask: (query) =>
    agent.ask({
      ...query,
      vault,
      context: {
        ...query.context,
        [CARD_CONTEXT.deck]: card.address.deck,
        [CARD_CONTEXT.card]: card.address.cardId,
        [CARD_CONTEXT.template]: card.address.template,
      },
    }),
  finish: (conversation) => agent.finish(conversation),
})
