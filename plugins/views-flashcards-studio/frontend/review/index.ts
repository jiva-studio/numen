/** The review of the cards: the window of the desktop.flashcards target, drawn whole. */
import { defineComponent, h, provide } from 'vue'
import type { NoticeHost, TabKindRegistration } from '@numen/plugin-sdk'
import Review from './app/Review.vue'
import { MOUNT_VIEW, type MountView } from './shared/views'

export { FLASHCARDS_TARGET } from './target'

/** The kind of the one tab the review is drawn in. */
export const REVIEW = 'flashcards.review'

/**
 * The review as a tab kind that opens in the main pane, saying what it has to say in notices and
 * mounting other plugins' views with `mountView`.
 */
export function createReviewKind(notices: NoticeHost, mountView: MountView): TabKindRegistration {
  return {
    kind: REVIEW,
    component: defineComponent({
      name: 'ReviewTab',
      setup() {
        provide(MOUNT_VIEW, mountView)
        return () => h(Review, { notices })
      },
    }),
    title: 'Flashcards',
    placement: { pane: 'main', order: 0 },
  }
}
