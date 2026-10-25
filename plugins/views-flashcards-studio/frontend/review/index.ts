/** The review of the cards: the window of the desktop.flashcards target, drawn whole. */
import type { TabKindRegistration } from '@numen/plugin-sdk'
import Review from './app/Review.vue'

export { FLASHCARDS_TARGET } from './target'

/** The kind of the one tab the review is drawn in. */
export const REVIEW = 'flashcards.review'

/** The review as a tab kind that opens in the main pane. */
export function createReviewKind(): TabKindRegistration {
  return {
    kind: REVIEW,
    component: Review,
    title: 'Flashcards',
    placement: { pane: 'main', order: 0 },
  }
}
