/**
 * The deck tab kind: a deck edited as a grid of cards, a tab to a file.
 */
import { Layers } from '@lucide/vue'
import type { TabKindRegistration } from '@numen/plugin-sdk'
import { getMarkOf } from '../buffer'
import type { Cards } from '../cards'
import DeckTab from './ui/DeckTab.vue'
import type { DeckTabState } from './types'

/** The word deck tabs are filed under. */
export const DECK = 'deck'

/** What the deck tabs of a window hold, as the kind that draws them reads it. */
export interface DeckTabsInside {
  readonly store: {
    open(id: string, path?: string): void
    getPath(id: string): string
    getOpenIds(): readonly string[]
    stale(id: string): unknown
    flushPending(id: string): Promise<void>
  }
  readonly vaultAnswers: { getTitle(path: string): string }
  readonly createDeck: Cards['createDeck']
  listCardModels(): Promise<void>
  listPresets(): Promise<void>
  listCardModelsAgain(): void
  listPresetsAgain(): void
  refreshDeckPreset(path: string): Promise<void>
  createDeckTabState(id: string): DeckTabState
}

export function createDeckKind({
  store,
  vaultAnswers,
  createDeck,
  listCardModels,
  listPresets,
  listCardModelsAgain,
  listPresetsAgain,
  refreshDeckPreset,
  createDeckTabState,
}: DeckTabsInside) {
  const kind: TabKindRegistration<DeckTabState> = {
    kind: DECK,
    component: DeckTab,
    icon: Layers,
    open: (path, id) => {
      store.open(id, path)
      void listCardModels()
      void listPresets()
      void refreshDeckPreset(path)
      return createDeckTabState(id)
    },
    // A deck already open under the path is the tab of that path, wherever the file has since moved.
    identity: (path) => {
      const open = store.getOpenIds().find((id) => store.getPath(id) === path)
      return open === undefined ? path : open.slice(DECK.length + 1)
    },
    getTitle: (one) => vaultAnswers.getTitle(store.getPath(one.id)),
    getMark: (one) => getMarkOf(one.note.value.state),
    onShow: () => {
      listCardModelsAgain()
      listPresetsAgain()
    },
    getTarget: (one) => ({
      path: store.getPath(one.id),
      title: vaultAnswers.getTitle(store.getPath(one.id)),
    }),
    getOpenTab: (one) => ({ path: store.getPath(one.id) }),
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    onDestroy: () => {},
    isAsking: (one) => store.stale(one.id) !== null,
    flushPending: (one) => store.flushPending(one.id),
    createFile: (folder, name) => createDeck(name, folder),
  }

  return kind
}
