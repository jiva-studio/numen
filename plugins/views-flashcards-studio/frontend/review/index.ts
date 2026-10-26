/** The review of the cards: the window of the desktop.flashcards target, drawn whole. */
import { defineComponent, h, provide } from 'vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import Review from './app/Review.vue'
import { MOUNT_VIEW, type MountView } from './shared/views'

export { FLASHCARDS_TARGET } from './target'

/** The kind of the one tab the review is drawn in. */
export const REVIEW = 'flashcards.review'

/** The review as a tab kind that opens in the main pane, mounting other plugins' views with `mountView`. */
export function createReviewKind(mountView: MountView): TabKindRegistration {
  return {
    kind: REVIEW,
    component: defineComponent({
      name: 'ReviewTab',
      setup() {
        provide(MOUNT_VIEW, mountView)
        return () => h(Review)
      },
    }),
    title: 'Flashcards',
    placement: { pane: 'main', order: 0 },
  }
}
