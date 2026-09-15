/**
 * Window registration for flashcard deck tabs.
 */
import { getMarkOf } from '@/entities/note'
import type { TabKind } from '@/entities/tab'
import { DECK } from '@/entities/tab'
import DeckTab from './ui/DeckTab.vue'
import type { DeckTabState } from './types'

/** What the deck tabs of a window hold, as the kind that draws them reads it. */
export interface DeckTabsInside {
  readonly cardTabPathMap: Map<string, string>
  readonly pendingTabIds: Map<string, string>
  readonly store: { open(id: string, path?: string): void; getPath(id: string): string }
  readonly vaultAnswers: { getTitle(path: string): string }
  listStencils(): Promise<void>
  listPresets(): Promise<void>
  listStencilsAgain(): void
  listPresetsAgain(): void
  refreshDeckPreset(path: string): Promise<void>
  createDeckTabState(id: string): DeckTabState
}

export function createDeckKind({
  cardTabPathMap,
  pendingTabIds,
  store,
  vaultAnswers,
  listStencils,
  listPresets,
  listStencilsAgain,
  listPresetsAgain,
  refreshDeckPreset,
  createDeckTabState,
}: DeckTabsInside) {
  const kind: TabKind<DeckTabState, typeof DECK> = {
    kind: DECK,
    open: (id) => {
      const path = cardTabPathMap.get(id) ?? id
      store.open(id, path)
      pendingTabIds.delete(path)
      cardTabPathMap.delete(id)
      void listStencils()
      void listPresets()
      void refreshDeckPreset(path)
      return createDeckTabState(id)
    },
    getTitle: (one) => vaultAnswers.getTitle(store.getPath(one.id)),
    getMark: (one) => getMarkOf(one.note.value.state),
    pane: DeckTab,
    identity: (id) => id,
    onShow: () => {
      listStencilsAgain()
      listPresetsAgain()
    },
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    onDestroy: () => {},
  }

  return kind
}
