/**
 * Window registration and tab state for flashcard deck tabs.
 */
import { computed } from 'vue'
import { asFailure, asValue } from '@numen/wire'
import type { BufferHost, PathRename } from '@numen/plugin-sdk'
import { openNotes } from '../../buffer'
import { getFailureCode, type Cards, type Presets } from '../../cards'
import { createDeckAnswers } from './answers'
import { createDeckReader } from './reader'
import { useDeckScheduleSync } from './useDeckScheduleSync'
import { createDeckTabActions } from './deckTabActions'
import { createDeckKind } from '../kind'
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

/** What the deck tabs of a window are made from. */
export interface DeckTabsOptions {
  readonly buffers: BufferHost
  /** The window lets go of a tab that took its own close. */
  readonly closeTab: (id: string) => void
}

export function useDeckTabs(cards: Cards, presets: Presets, options: DeckTabsOptions) {
  const { buffers, closeTab } = options
  const vaultAnswers = createDeckAnswers()
  const store = createDeckNoteStore(cards, vaultAnswers)
  buffers.track(store)
  const readItem = createDeckReader(store, vaultAnswers.getProblems)
  const wiring = useDeckScheduleSync(cards, presets, store)
  const { createDeckTabState } = createDeckTabFactory({
    store,
    read: readItem,
    vaultAnswers,
    wiring,
    closeTab,
  })

  const kind = createDeckKind({
    store,
    vaultAnswers,
    createDeck: cards.createDeck,
    listCardModels: wiring.listCardModels,
    listPresets: wiring.scheduled.listing.listPresets,
    listCardModelsAgain: wiring.listCardModelsAgain,
    listPresetsAgain: wiring.scheduled.listing.listPresetsAgain,
    refreshDeckPreset: wiring.scheduled.refreshDeckPreset,
    createDeckTabState,
  })

  const applyPathChanges = (
    paths: readonly string[],
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    renames: readonly PathRename[] = [],
  ): void => {
    for (const rename of renames) vaultAnswers.moveFile(rename.from, rename.to)
    store.applyPathChanges(paths, renames)
    wiring.applyPathChanges(paths, renames)
  }

  return {
    kind,
    createDeckTabState,
    applyPathChanges,
    listCardModels: wiring.listCardModels,
    getTitle: vaultAnswers.getTitle,
    store,
  }
}

function createDeckNoteStore(
  cards: Cards,
  vaultAnswers: ReturnType<typeof createDeckAnswers>,
): ReturnType<typeof openNotes> {
  return openNotes({
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
      const readItem = answer.value.deck
      vaultAnswers.recordRead(path, {
        problems: readItem.problems,
        error: null,
        bound: 0,
        title: readItem.title,
      })
      return asValue({
        body: serializeBufferDeckToString(deserializeVaultDeck(readItem)),
        at: answer.value.at,
      })
    },
    write: async (path, body, observed) => {
      const deck = deserializeBufferDeckFromString(body)
      const answer = await cards.writeDeck(
        path,
        {
          preamble: deck.preamble,
          cards: serializeBufferCardsToVaultCards(deck),
          sections: serializeBufferSectionsToVaultSections(deck),
          tail: deck.tail,
        },
        observed?.fingerprint ?? null,
      )
      vaultAnswers.recordWrite(path, {
        error: answer.ok ? null : getFailureCode(answer.error.code),
        bound: answer.ok ? 0 : answer.error.bound,
      })
      if (!answer.ok) return asFailure(answer.error.code)
      return asValue({ body: '', at: answer.value.at })
    },
  })
}

interface DeckTabStateFactoryDeps {
  store: ReturnType<typeof openNotes>
  read: ReturnType<typeof createDeckReader>
  vaultAnswers: ReturnType<typeof createDeckAnswers>
  wiring: ReturnType<typeof useDeckScheduleSync>
  closeTab: (id: string) => void
}

function createDeckTabFactory(deps: DeckTabStateFactoryDeps) {
  const { store, read, vaultAnswers, wiring, closeTab } = deps
  const { getDeck, getMarks } = read
  const { offers, cardModels, scheduled } = wiring
  const { listing, scheduleDeck, getDeckPreset } = scheduled
  const { choices } = listing

  const updateDeckState = (id: string, deck: BufferDeck): void => {
    const body = serializeBufferDeckToString(deck)
    read.setParsed(id, body, deck)
    store.setBody(id, body)
  }

  const forgetPath = (path: string): void => {
    if (store.getOpenIds().some((one) => store.getPath(one) === path)) return
    vaultAnswers.forgetFile(path)
    scheduled.forgetFile(path)
  }

  const createDeckTabState = (id: string): DeckTabState => {
    const deck = computed(() => getDeck(id))
    const actions = createDeckTabActions(id, {
      getDeck,
      updateDeck: updateDeckState,
      getOffers: () => offers.value,
    })

    return {
      id,
      note: computed(() => store.getOpenNote(id)),
      deck,
      deckCards: computed(() => createDeckCards(deck.value, offers.value)),
      sections: computed(() => createDeckSections(deck.value)),
      cardModels,
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
        void store.close(id).then((removed) => {
          if (!removed) return
          read.forgetTab(id)
          scheduled.forgetTab(id)
          forgetPath(path)
          closeTab(tab)
        })
      },
    }
  }

  return { createDeckTabState, updateDeckState, forgetPath }
}
