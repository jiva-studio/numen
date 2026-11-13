/**
 * The deck tab kind: a deck edited as a grid of cards, a tab to a file.
 */
import { Layers } from '@lucide/vue'
import type { Component } from 'vue'
import type { TabKindRegistration, Caught } from '@numen/plugin-sdk'
import { getMarkOf } from '../../lib/buffer/words'
import type { Cards } from '../../../types'
import type { ConflictWords } from '../../lib/buffer/types'
import type { DeckTabState } from './tabState'

/** The word deck tabs are filed under. */
export const DECK = 'deck'

/** The lists a deck tab loads when it opens and reloads when it is shown. */
interface DeckTabLists {
  listCardModels(): Promise<void>
  listPresets(): Promise<void>
  listCardModelsAgain(): void
  listPresetsAgain(): void
  refreshDeckPreset(path: string): Promise<void>
}

/** What the deck tabs of a window hold, as the kind that draws them reads it. */
export interface DeckTabsInside {
  readonly store: {
    open(id: string, path?: string): void
    getPath(id: string): string
    getOpenIds(): readonly string[]
    stale(id: string): ConflictWords | null
    flushPending(id: string): Promise<void>
  }
  readonly vaultAnswers: { getTitle(path: string): string }
  readonly createDeck: Cards['createDeck']
  readonly lists: DeckTabLists
  /** Shows the failure of an operation that runs in the background. */
  reportFailure(error: Caught): void
  createDeckTabState(id: string): DeckTabState
  readonly component: Component
}

export function createDeckKind({
  store,
  vaultAnswers,
  createDeck,
  lists,
  reportFailure,
  createDeckTabState,
  component,
}: DeckTabsInside) {
  const kind: TabKindRegistration<DeckTabState> = {
    kind: DECK,
    component,
    icon: Layers,
    open: (path, id) => {
      store.open(id, path)
      lists.listCardModels().catch(reportFailure)
      lists.listPresets().catch(reportFailure)
      lists.refreshDeckPreset(path).catch(reportFailure)
      return createDeckTabState(id)
    },
    // A deck already open under the path is the tab of that path, wherever the file has since moved.
    identity: (path) => {
      const open = store.getOpenIds().find((id) => store.getPath(id) === path)
      return open === undefined ? path : open.slice(DECK.length + 1)
    },
    getTitle: (one) => vaultAnswers.getTitle(store.getPath(one.id)),
    getMark: (one) => getMarkOf(one.data.note.value.state),
    onShow: () => {
      lists.listCardModelsAgain()
      lists.listPresetsAgain()
    },
    getTarget: (one) => ({
      path: store.getPath(one.id),
      title: vaultAnswers.getTitle(store.getPath(one.id)),
    }),
    getOpenTab: (one) => ({ path: store.getPath(one.id) }),
    onClose: (one, id) => {
      one.file.close(id)
      return false
    },
    // The tab state holds no resource beyond the buffer store.
    onDestroy: () => undefined,
    isAsking: (one) => store.stale(one.id) !== null,
    flushPending: (one) => store.flushPending(one.id),
    createFile: (folder, name) => createDeck(name, folder),
  }

  return { kind }
}
