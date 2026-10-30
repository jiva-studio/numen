/**
 * The card model tab kind and the state of its tabs: a card model edited as its fields and templates.
 */
import { computed } from 'vue'
import { LayoutTemplate } from '@lucide/vue'
import {
  formatErrorMessage,
  getFileName,
  type BufferHost,
  type MessageWriter,
  type PathRename,
  type TabKindRegistration,
} from '@numen/plugin-sdk'
import { getMarkOf, openNotes } from '../buffer'
import { WORDS as words, type Cards } from '../cards'
import CardModelTab from './ui/CardModelTab.vue'
import type { CardModelTabState } from './types'
import { createCardModelWire, type VaultAnswer } from './api/wire'
import { createCardModelFields } from './model/fields'

export type { CardModelTabState, VaultAnswer }

/** The word card model tabs are filed under. */
export const CARD_MODEL = 'card-model'

/** What the card model tabs of a window are made from. */
export interface CardModelTabsOptions {
  readonly buffers: BufferHost
  /** The window lets go of a tab that took its own close. */
  readonly closeTab: (id: string) => void
  readonly writeMessage?: MessageWriter | undefined
}

type CardModelFields = ReturnType<typeof createCardModelFields>
type CardModelWire = ReturnType<typeof createCardModelWire>

function createCardModelTabItem(
  id: string,
  deps: {
    store: ReturnType<typeof openNotes>
    fields: CardModelFields
    wire: CardModelWire
    closeTab: (id: string) => void
    reportFailure: (error: unknown) => void
  },
): CardModelTabState {
  const { store, fields, wire, closeTab, reportFailure } = deps
  const close = (tab: string) => {
    const path = store.getPath(id)
    store
      .close(id)
      .then((removed) => {
        if (!removed) return
        fields.forget(id)
        wire.forget(
          path,
          store.getOpenIds().some((one) => store.getPath(one) === path),
        )
        closeTab(tab)
      })
      .catch(reportFailure)
  }

  return {
    id,
    note: computed(() => store.getOpenNote(id)),
    cardModel: computed(() => fields.getCardModel(id)),
    marks: computed(() => fields.getMarks(id)),
    errorMessage: computed(() =>
      wire.getErrorMessage(store.getPath(id), store.getOpenNote(id).error),
    ),
    ...fields.createActions(id),
    keepMine: () => store.keep(id),
    takeFile: () => store.take(id),
    close,
  }
}

function createCardModelKindDefinition(deps: {
  store: ReturnType<typeof openNotes>
  wire: CardModelWire
  cards: Cards
  createCardModelTabState: (id: string) => CardModelTabState
}): TabKindRegistration<CardModelTabState> {
  const { store, wire, cards } = deps
  const getTitle = (path: string): string => wire.getTitle(path) || getFileName(path)
  return {
    kind: CARD_MODEL,
    component: CardModelTab,
    icon: LayoutTemplate,
    open: (path, id) => {
      store.open(id, path)
      return deps.createCardModelTabState(id)
    },
    // A card model already open under the path is the tab of that path, wherever the file has since moved.
    identity: (path) => {
      const open = store.getOpenIds().find((id) => store.getPath(id) === path)
      return open === undefined ? path : open.slice(CARD_MODEL.length + 1)
    },
    getTitle: (one) => getTitle(store.getPath(one.id)),
    getMark: (one) => getMarkOf(one.note.value.state),
    getTarget: (one) => ({
      path: store.getPath(one.id),
      title: getTitle(store.getPath(one.id)),
    }),
    getOpenTab: (one) => ({ path: store.getPath(one.id) }),
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    // The tab state holds no resource beyond the buffer store.
    onDestroy: () => undefined,
    isAsking: (one) => store.stale(one.id) !== null,
    flushPending: (one) => store.flushPending(one.id),
    createFile: (folder, name) => cards.createCardModel(name, folder, [words.newField]),
  }
}

export function createCardModelTabs(cards: Cards, options: CardModelTabsOptions) {
  // Messages are dropped when the caller supplies no writer.
  const { buffers, closeTab, writeMessage = () => undefined } = options
  const reportFailure = (error: unknown): void => writeMessage(formatErrorMessage(error), 'error')
  const wire = createCardModelWire(cards, writeMessage)
  const store = openNotes({ read: wire.read, write: wire.write })
  buffers.track(store)
  const fields = createCardModelFields(
    (id) => store.getOpenNote(id).body,
    (id, body) => store.setBody(id, body),
    (id) => wire.getProblems(store.getPath(id)),
    (id, field, name) => {
      wire
        .renameField(store.getPath(id), { from: field, to: name }, store.applyPathChanges)
        .catch(reportFailure)
    },
  )

  const createCardModelTabState = (id: string): CardModelTabState =>
    createCardModelTabItem(id, { store, fields, wire, closeTab, reportFailure })
  const kind = createCardModelKindDefinition({ store, wire, cards, createCardModelTabState })

  return {
    kind,
    createCardModelTabState,
    applyPathChanges: (paths: readonly string[], renames: readonly PathRename[] = []) => {
      wire.movePaths(renames)
      store.applyPathChanges(paths, renames)
    },
    getTitle: (path: string): string => wire.getTitle(path) || getFileName(path),
    store,
  }
}

export const useCardModelTabs = createCardModelTabs
