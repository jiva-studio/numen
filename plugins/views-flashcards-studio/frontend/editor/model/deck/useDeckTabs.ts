/**
 * Window registration and tab state for flashcard deck tabs.
 */
import { computed, type Component } from 'vue'
import {
  asFailure,
  asValue,
  formatErrorMessage,
  type BufferHost,
  type MessageWriter,
  type PathRename,
} from '@numen/plugin-sdk'
import { openNotes } from '../buffer/notes'
import { getFailureCode } from '../../../lib/failures'
import type { Cards } from '../../../api/cards'
import type { Presets } from '../../../lib/presets'
import { createDeckAnswers } from './answers'
import { createDeckReader } from './reader'
import { useDeckScheduleSync } from './useDeckScheduleSync'
import { createDeckTabActions } from './deckTabActions'
import { createDeckKind } from './kind'
import {
  createDeckSections,
  serializeBufferDeckToString,
  serializeBufferCardsToVaultCards,
  deserializeBufferDeckFromString,
  deserializeVaultDeck,
  createDeckCards,
  serializeBufferSectionsToVaultSections,
  type BufferDeck,
} from '../../lib/deck/deck'
import type { DeckTabState } from '../../lib/deck/types'

export type { DeckTabState }

/** What the deck tabs of a window are made from. */
export interface DeckTabsOptions {
  readonly buffers: BufferHost
  readonly closeTab: (id: string) => void
  readonly writeMessage?: MessageWriter | undefined
  readonly component: Component
}

export function useDeckTabs(cards: Cards, presets: Presets, options: DeckTabsOptions) {
  // Messages are dropped when the caller supplies no writer.
  const { buffers, closeTab, writeMessage = () => undefined, component } = options
  const reportFailure = (error: unknown): void => writeMessage(formatErrorMessage(error), 'error')
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
    reportFailure,
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
    reportFailure,
    createDeckTabState,
    component,
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
        observed?.stat ?? null,
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
  reportFailure: (error: unknown) => void
}

function createDeckTabFactory(deps: DeckTabStateFactoryDeps) {
  const { store, read, vaultAnswers, wiring, closeTab, reportFailure } = deps
  const { getDeck, getMarks } = read
  const { summaries, cardModels, scheduled } = wiring
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
      getCardModelChoices: () => summaries.value,
    })

    return {
      id,
      data: {
        note: computed(() => store.getOpenNote(id)),
        deck,
        deckCards: computed(() => createDeckCards(deck.value, summaries.value)),
        sections: computed(() => createDeckSections(deck.value)),
        cardModels,
        marks: computed(() => getMarks(id)),
        errorMessage: computed(() =>
          vaultAnswers.getErrorMessage(store.getPath(id), store.getOpenNote(id).error !== null),
        ),
      },
      schedule: {
        scheduled: computed(() => getDeckPreset(id)),
        choices,
        setSchedule: (preset) => {
          scheduleDeck(id, preset).catch(reportFailure)
        },
      },
      actions,
      file: {
        keepMine: () => store.keep(id),
        takeFile: () => store.take(id),
        close: (tab) => {
          const path = store.getPath(id)
          store
            .close(id)
            .then((removed) => {
              if (!removed) return
              read.forgetTab(id)
              scheduled.forgetTab(id)
              forgetPath(path)
              closeTab(tab)
            })
            .catch(reportFailure)
        },
      },
    }
  }

  return { createDeckTabState, updateDeckState, forgetPath }
}
