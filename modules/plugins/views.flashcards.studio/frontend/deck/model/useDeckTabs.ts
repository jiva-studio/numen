/**
 * Window registration and tab state for flashcard deck tabs.
 */
import { computed } from 'vue'
import { asFailure, asValue } from '@numen/wire'
import { openNotes, type BufferHost, type FileBuffers, type PathRename } from '@numen/plugin-sdk'
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
  const read = createDeckReader(store, vaultAnswers.getProblems)
  const wiring = useDeckScheduleSync(cards, presets, store)
  const { createDeckTabState } = createDeckTabFactory({
    store,
    read,
    vaultAnswers,
    wiring,
    closeTab,
  })

  const kind = createDeckKind({
    store,
    vaultAnswers,
    createDeck: cards.createDeck,
    listStencils: wiring.listStencils,
    listPresets: wiring.scheduled.listPresets,
    listStencilsAgain: wiring.listStencilsAgain,
    listPresetsAgain: wiring.scheduled.listPresetsAgain,
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
    listStencils: wiring.listStencils,
    getTitle: vaultAnswers.getTitle,
    store,
  }
}

function createDeckNoteStore(
  cards: Cards,
  vaultAnswers: ReturnType<typeof createDeckAnswers>,
): FileBuffers {
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
  })
}

interface DeckTabStateFactoryDeps {
  store: FileBuffers
  read: ReturnType<typeof createDeckReader>
  vaultAnswers: ReturnType<typeof createDeckAnswers>
  wiring: ReturnType<typeof useDeckScheduleSync>
  closeTab: (id: string) => void
}

function createDeckTabFactory(deps: DeckTabStateFactoryDeps) {
  const { store, read, vaultAnswers, wiring, closeTab } = deps
  const { getDeck, getMarks } = read
  const { offers, stencils, scheduled } = wiring
  const { choices, scheduleDeck, getDeckPreset } = scheduled

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
          closeTab(tab)
        })
      },
    }
  }

  return { createDeckTabState, updateDeckState, forgetPath }
}
