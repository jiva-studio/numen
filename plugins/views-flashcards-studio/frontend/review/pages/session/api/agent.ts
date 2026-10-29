/**
 * Questions about the card currently displayed.
 *
 * Nothing here renders: the port is the window's `AgentPort`, and this module
 * adds the card the question is about.
 */
import type { AgentPort } from '@numen/plugin-sdk'
import type { ReviewCard } from '../../../entities/card'

/** The keys under which the card tools read the card currently displayed. */
export const CARD_CONTEXT = {
  deck: 'md.numen.views.flashcards.studio/deck',
  card: 'md.numen.views.flashcards.studio/card',
  template: 'md.numen.views.flashcards.studio/template',
} as const

/**
 * The agent for a conversation about one card of a vault.
 *
 * The card identity is passed in the context of the question and not inside
 * it. A deck name in a synced vault is set by the sync source, and a name in
 * the question text would be read as an instruction; `card_current` returns the
 * same name as a tool result, which is data. The window has no vault of its own,
 * so the question names the vault the card tools operate on.
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
        [CARD_CONTEXT.deck]: card.deck,
        [CARD_CONTEXT.card]: card.cardId,
        [CARD_CONTEXT.template]: card.template,
      },
    }),
  finish: (conversation) => agent.finish(conversation),
})
