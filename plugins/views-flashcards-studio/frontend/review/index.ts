/**
 * The review of the cards in the window of the desktop.flashcards target: a tab
 * that opens on the vault the user chose, and the summary rendered beside each
 * vault on the window's list.
 */
import { computed, defineComponent, h, type PropType } from 'vue'
import {
  VAULT_SUMMARY,
  type AgentHost,
  type NoticeHost,
  type TabKindRegistration,
  type VaultSummaryContext,
  type ViewRegistration,
} from '@numen/plugin-sdk'
import ReviewTab from './ui/ReviewTab.vue'
import { useReview, type Review } from './model/review'
import type { VaultCounts } from './model/counts'
import VaultSummary from './ui/vaults/VaultSummary.vue'
import type { MountView } from './model/views'

export { createVaultCounts } from './model/counts'
export { createNotices } from './model/notices'

/** The tab kind the review is rendered in. */
export const REVIEW = 'flashcards.review'

/** The dependencies the review needs from the window it is rendered in. */
export interface ReviewKindDeps {
  readonly counts: VaultCounts
  readonly notices: NoticeHost
  readonly agent: AgentHost
  readonly mountView: MountView
  readonly closeTab: (id: string) => void
}

/**
 * The review as a tab kind that opens in the main pane on a vault, saying what it has to say in
 * notices and mounting other plugins' views with `mountView`.
 */
export function createReviewKind(deps: ReviewKindDeps): TabKindRegistration<Review> {
  return {
    kind: REVIEW,
    component: ReviewTab,
    title: 'Flashcards',
    placement: { pane: 'main', order: 0 },
    open: (vault, id) =>
      useReview({
        vault,
        counts: deps.counts,
        notices: deps.notices,
        agent: deps.agent,
        mountView: deps.mountView,
        goToVaults: () => deps.closeTab(id),
        getViewport: () => ({
          screen: window.screen.width,
          page: document.documentElement.clientWidth,
        }),
      }),
    getTitle: (state) => state.vaultCount.value?.identity.name ?? 'Flashcards',
    onKeyPress: (state, event) => state.onKeyDown(event),
  }
}

/** The cards due in each vault, as the view a window mounts beside it on its list. */
export function createVaultSummaryView(counts: VaultCounts): ViewRegistration {
  return {
    capability: VAULT_SUMMARY,
    component: defineComponent({
      name: 'VaultSummaryView',
      props: { context: { type: Object as PropType<VaultSummaryContext>, required: true } },
      setup(props) {
        counts.start()
        const vault = computed(
          () =>
            counts.vaults.value.find((one) => one.identity.vault === props.context.vault) ?? null,
        )
        return () => h(VaultSummary, { vault: vault.value })
      },
    }),
  }
}
