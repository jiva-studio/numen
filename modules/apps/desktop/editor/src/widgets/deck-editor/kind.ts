/**
 * Window registration for flashcard deck tabs.
 */
import { markOf } from '@/entities/note'
import type { TabKind } from '@/entities/tab'
import { DECK } from '@/entities/tab'
import DeckTab from './ui/DeckTab.vue'
import type { DeckTabState } from './types'

/** What the deck tabs of a window hold, as the kind that draws them reads it. */
export interface DeckTabsInside {
  readonly cardTabPathMap: Map<string, string>
  readonly pendingTabIds: Map<string, string>
  readonly store: { open(id: string, path?: string): void; where(id: string): string }
  readonly said: { called(path: string): string }
  lists(): Promise<void>
  listsPresets(): Promise<void>
  listsAgain(): void
  listsPresetsAgain(): void
  asks(path: string): Promise<void>
  createDeckTabState(id: string): DeckTabState
}

export function deckKind({
  cardTabPathMap,
  pendingTabIds,
  store,
  said,
  lists,
  listsPresets,
  listsAgain,
  listsPresetsAgain,
  asks,
  createDeckTabState,
}: DeckTabsInside) {
  const kind: TabKind<DeckTabState, typeof DECK> = {
    kind: DECK,
    opens: (id) => {
      const path = cardTabPathMap.get(id) ?? id
      store.open(id, path)
      pendingTabIds.delete(path)
      cardTabPathMap.delete(id)
      void lists()
      void listsPresets()
      void asks(path)
      return createDeckTabState(id)
    },
    called: (one) => said.called(store.where(one.id)),
    marked: (one) => markOf(one.shown.value.state),
    draws: DeckTab,
    identity: (id) => id,
    shown: () => {
      listsAgain()
      listsPresetsAgain()
    },
    shuts: (one, id) => {
      one.close(id)
      return false
    },
    gone: () => {},
  }

  return kind
}
