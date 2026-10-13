/**
 * Window registration and tab state for flashcard deck tabs.
 */
import { computed } from 'vue'
import { asFailure, asValue } from '@numen/wire'
import type { PlexDestination } from '@numen/ui'
import type { PathRename } from '@/shared/paths'
import { getFailureCode } from '@/entities/deck'
import type { Cards } from '@/entities/deck'
import type { Store } from '@/features/command-palette'
import type { Presets } from '@/entities/deck'
import { createDeckAnswers } from './answers'
import { createDeckReader } from './reader'
import { useDeckScheduleSync } from './useDeckScheduleSync'
import { createDeckTabActions } from './deckTabActions'
import { createDeckKind } from '../kind'
import { openNotes } from '@/entities/note'
import type { WindowHandle } from '@/entities/tab'
import type { FileOpeners } from '@/entities/tab'
import { DECK } from '@/entities/tab'
import {
  createDeckSections,
  serializeBufferDeckToString,
  serializeBufferCardsToVaultCards,
  deserializeBufferDeckFromString,
  deserializeVaultDeck,
  createDeckCards,
  serializeBufferSectionsToVaultSections,
  type BufferDeck,
} from '../lib/deck'
import type { DeckTabState } from '../types'

export type { DeckTabState }

export interface DeckTabsDeps {
  now?: () => number
}

export function useDeckTabs(
  cards: Cards,
  presets: Presets,
  handle: WindowHandle,
  tabOpeners: FileOpeners,
  deps?: DeckTabsDeps,
) {
  const vaultAnswers = createDeckAnswers()
  const now = deps?.now ?? (() => 0)

  const store = openNotes(
    {
      read: async (path) => {
        const answer = await cards.readDeck(path)
        if (!answer.ok) {
          const code = getFailureCode(answer.error.code)
          vaultAnswers.recordRead(path, {
            problems: [],
            error: code,
            bound: answer.error.bound,
            title: null,
          })
          return asFailure(code ?? 'notADeck')
        }
        const read = answer.value.deck
        vaultAnswers.recordRead(path, {
          problems: read.problems,
          error: null,
          bound: 0,
          title: read.title,
        })
        return asValue({
          body: serializeBufferDeckToString(deserializeVaultDeck(read)),
          at: answer.value.at,
        })
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
          seen?.fingerprint ?? null,
        )
        vaultAnswers.recordWrite(path, {
          error: answer.ok ? null : getFailureCode(answer.error.code),
          bound: answer.ok ? 0 : answer.error.bound,
        })
        if (!answer.ok) return asFailure(answer.error.code)
        return asValue({ body: '', at: answer.value.at })
      },
    },
    { now },
  )

  const read = createDeckReader(store, vaultAnswers.getProblems)
  const { getDeck, getMarks } = read

  const wiring = useDeckScheduleSync(cards, presets, store)
  const { offers, stencils, listStencils, listStencilsAgain, scheduled } = wiring
  const { choices, listPresets, listPresetsAgain, refreshDeckPreset, scheduleDeck, getDeckPreset } =
    scheduled

  const updateDeckState = (id: string, deck: BufferDeck): void => {
    const body = serializeBufferDeckToString(deck)
    read.setParsed(id, body, deck)
    store.setBody(id, body)
  }

  const createDeckTabState = (id: string): DeckTabState => {
    const deck = computed(() => getDeck(id))
    const actions = createDeckTabActions(id, getDeck, updateDeckState, () => offers.value)

    return {
      id,
      note: computed(() => store.getOpenNote(id)),
      deck,
      drawn: computed(() => createDeckCards(deck.value, offers.value)),
      sections: computed(() => createDeckSections(deck.value)),
      stencils,
      marks: computed(() => getMarks(id)),
      errorMessage: computed(() =>
        vaultAnswers.getErrorMessage(store.getPath(id), store.getOpenNote(id).error !== null),
      ),
      scheduled: computed(() => getDeckPreset(id)),
      choices,
      setSchedule: (preset) => void scheduleDeck(id, preset),
      ...actions,
      keepMine: () => store.keep(id),
      takeFile: () => store.take(id),
      close: (tab) => {
        const path = store.getPath(id)
        void store.close(id).then((gone) => {
          if (!gone) return
          read.forgetTab(id)
          scheduled.forgetTab(id)
          forgetPath(path)
          handle.closeTab(tab)
        })
      },
    }
  }

  const forgetPath = (path: string): void => {
    if (store.getOpenIds().some((one) => store.getPath(one) === path)) return
    vaultAnswers.forgetFile(path)
    scheduled.forgetFile(path)
  }

  const closeTabById = (id: string): void => {
    const tab = handle.each<DeckTabState>(DECK).find((one) => one.state.id === id)
    tab?.state.close(tab.id)
  }

  const kept: Store = {
    has: (id) => store.getOpenIds().includes(id),
    getPath: (id) => store.getPath(id),
    getTitle: (id) => vaultAnswers.getTitle(store.getPath(id)),
    asking: (id) => store.stale(id) !== null,
    flushPending: (id) => store.flushPending(id),
    close: closeTabById,
    getTabAt: (path) => store.getOpenIds().find((id) => store.getPath(id) === path) ?? null,
  }

  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(store.getOpenIds().map((one) => [store.getPath(one), one])),
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

  const kind = createDeckKind({
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
  })

  const openDeckTab = (path: string, title: string, how: PlexDestination): void => {
    const id = getOrCreateTabId(path)
    if (title) vaultAnswers.setTitle(path, title)
    void (how === 'beside' ? handle.openTabBeside(DECK, id) : handle.openTab(DECK, id))
  }

  tabOpeners.registerEditor('deck', openDeckTab)

  const applyPathChanges = (
    paths: readonly string[],
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    renames: readonly PathRename[] = [],
  ): void => {
    for (const rename of renames) {
      vaultAnswers.moveFile(rename.from, rename.to)
    }
    store.applyPathChanges(paths, renames)
    wiring.applyPathChanges(paths, renames)
  }

  return {
    kind,
    createDeckTabState,
    applyPathChanges,
    listStencils,
    getTitle: vaultAnswers.getTitle,
    kept,
    getOpenIds: store.getOpenIds,
    getOpenNote: store.getOpenNote,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
