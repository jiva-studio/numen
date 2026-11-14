/**
 * The card a question is about, as the tools the agent calls read it.
 */
import type { ReviewCard } from '../../lib/card/cards'

/** The keys under which the card tools read the card currently displayed. */
const CARD_CONTEXT = {
  deck: 'md.numen.views.flashcards.studio/deck',
  card: 'md.numen.views.flashcards.studio/card',
  template: 'md.numen.views.flashcards.studio/template',
} as const

/**
 * The tool context of a conversation about one card. The card identity goes to the tools, not into
 * the question: a deck name in a synced vault is set by the sync source, and in the question it
 * would be read as an instruction, while `card_current_get` returns it as a tool result, which is
 * data.
 */
export const getCardToolContext = (card: ReviewCard): Readonly<Record<string, string>> => ({
  [CARD_CONTEXT.deck]: card.address.deck,
  [CARD_CONTEXT.card]: card.address.cardId,
  [CARD_CONTEXT.template]: card.address.template,
})
