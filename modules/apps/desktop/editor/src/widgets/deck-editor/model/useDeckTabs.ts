/**
 * Window registration and tab state for flashcard deck tabs.
 */
import { computed } from 'vue'
import type { PlexShowing } from '@numen/ui'
import type { PathRename } from '@/shared/paths'
import type { Cards } from '@/entities/deck/cards'
import type { Store } from '@/features/command-palette/deps'
import type { Presets } from '@/entities/deck/presets'
import { answers } from '../answers'
import { reader } from '../reader'
import { useDeckScheduleSync } from './useDeckScheduleSync'
import { createDeckTabActions } from '../deckTabActions'
import { deckKind } from '../kind'
import { openNotes } from '@/entities/note'
import type { WindowHandle } from '@/entities/tab/windowTabs'
import type { FileOpeners } from '@/entities/tab/openers'
import { DECK } from '@/entities/tab/workspace'
import {
  drawnSectionsOf,
  serializeBufferDeckToString,
  serializeBufferCardsToVaultCards,
  deserializeBufferDeckFromString,
  deserializeVaultDeck,
  drawnOf,
  serializeBufferSectionsToVaultSections,
  type BufferDeck,
} from '../deck'
import type { DeckTabState } from '../types'

export type { DeckTabState }

export function useDeckTabs(
  cards: Cards,
  presets: Presets,
  handle: WindowHandle,
  puts: FileOpeners,
) {
  const said = answers()

  const store = openNotes({
    read: async (path) => {
      const answer = await cards.readDeck(path)
      const deck = answer.deck ? deserializeVaultDeck(answer.deck) : null
      const error = answer.error
      said.reads(path, {
        problems: answer.deck?.problems ?? [],
        error,
        bound: answer.bound,
        title: answer.deck?.title ?? null,
      })
      if (error !== null) return { body: '', error }
      return { body: deck ? serializeBufferDeckToString(deck) : '', error: null, at: answer.at }
    },
    write: async (path, body, seen) => {
      const deck = deserializeBufferDeckFromString(body)
      const answer = await cards.writeDeck(
        path,
        {
          preamble: deck.preamble,
          cards: serializeBufferCardsToVaultCards(deck),
          sections: serializeBufferSectionsToVaultSections(deck),
          tail: deck.tail,
        },
        seen?.at ?? null,
      )
      const error = answer.error
      said.writes(path, { error, bound: answer.bound })
      return {
        body: '',
        error,
        at: answer.at,
        changed: answer.changed,
      }
    },
  })

  const read = reader(store, said.problemsAt)
  const { deckAt, marksAt } = read

  const wiring = useDeckScheduleSync(cards, presets, store)
  const { offers, stencils, lists, listsAgain, scheduled } = wiring
  const { choices, listsPresets, listsPresetsAgain, asks, schedules, scheduledAt } = scheduled

  const updateDeckState = (id: string, deck: BufferDeck): void => {
    const body = serializeBufferDeckToString(deck)
    read.holds(id, body, deck)
    store.typed(id, body)
  }

  const createDeckTabState = (id: string): DeckTabState => {
    const deck = computed(() => deckAt(id))
    const actions = createDeckTabActions(id, deckAt, updateDeckState, () => offers.value)

    return {
      id,
      shown: computed(() => store.shown(id)),
      deck,
      drawn: computed(() => drawnOf(deck.value, offers.value)),
      sections: computed(() => drawnSectionsOf(deck.value)),
      stencils,
      marks: computed(() => marksAt(id)),
      errorMessage: computed(() =>
        said.getErrorMessage(store.where(id), store.shown(id).error !== null),
      ),
      scheduled: computed(() => scheduledAt(id)),
      choices,
      setSchedule: (preset) => void schedules(id, preset),
      ...actions,
      keepMine: () => store.keep(id),
      takeFile: () => store.take(id),
      close: (tab) => {
        const path = store.where(id)
        void store.shut(id).then((gone) => {
          if (!gone) return
          read.closes(id)
          scheduled.closes(id)
          forgetPath(path)
          handle.closes(tab)
        })
      },
    }
  }

  const forgetPath = (path: string): void => {
    if (store.all().some((one) => store.where(one) === path)) return
    said.forgets(path)
    scheduled.forgets(path)
  }

  const closeTabById = (id: string): void => {
    const tab = handle.each<DeckTabState>(DECK).find((one) => one.state.id === id)
    tab?.state.close(tab.id)
  }

  const kept: Store = {
    has: (id) => store.all().includes(id),
    where: (id) => store.where(id),
    called: (id) => said.called(store.where(id)),
    asking: (id) => store.stale(id) !== null,
    settles: (id) => store.settles(id),
    shuts: closeTabById,
    holding: (path) => store.all().find((id) => store.where(id) === path) ?? null,
  }

  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(store.all().map((one) => [store.where(one), one])),
  )

  const pendingTabIds = new Map<string, string>()
  const cardTabPathMap = new Map<string, string>()

  const getOrCreateTabId = (path: string): string => {
    const open = tabbed.value.get(path) ?? pendingTabIds.get(path)
    if (open) return open
    const one = crypto.randomUUID()
    pendingTabIds.set(path, one)
    cardTabPathMap.set(one, path)
    return one
  }

  const kind = deckKind({
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
  })

  const openDeckTab = (path: string, title = '', showing: PlexShowing = 'here'): void => {
    const id = getOrCreateTabId(path)
    if (title) said.names(path, title)
    void (showing === 'beside' ? handle.beside(DECK, id) : handle.opens(DECK, id))
  }

  puts.holds('deck', openDeckTab)

  const changed = (paths: readonly string[], renamed: readonly PathRename[] = []): void => {
    for (const went of renamed) {
      said.moved(went.from, went.to)
    }
    store.changed(paths, renamed)
    wiring.changed(paths, renamed)
  }

  return {
    kind,
    held: createDeckTabState,
    changed,
    lists,
    called: said.called,
    kept,
    all: store.all,
    shown: store.shown,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
