/**
 * The card model tab kind and the state of its tabs: a card model edited as its fields and templates.
 */
import { computed, type Component } from 'vue'
import { LayoutTemplate } from '@lucide/vue'
import {
  formatErrorMessage,
  getFileName,
  type BufferHost,
  type MessageWriter,
  type PathRename,
  type TabKindRegistration,
  type Caught,
} from '@numen/plugin-sdk'
import { getMarkOf } from '../../lib/buffer/words'
import { openNotes } from '../buffer/notes'
import { WORDS as words } from '../../../words'
import {
  createCardModelState,
  createCardModelWire,
  type CardModelState,
} from '../../api/card-model/wire'
import { createCardModelFields } from './fields'
import type { Cards } from '../../../types'
import type { CardModelTabState } from './types'

export type { CardModelTabState }

/** The word card model tabs are filed under. */
export const CARD_MODEL = 'card-model'

/** What the card model tabs of a window are made from. */
export interface CardModelTabsOptions {
  readonly buffers: BufferHost
  readonly closeTab: (id: string) => void
  readonly writeMessage?: MessageWriter | undefined
  readonly component: Component
}

type CardModelFields = ReturnType<typeof createCardModelFields>

function createCardModelTabItem(
  id: string,
  deps: {
    store: ReturnType<typeof openNotes>
    fields: CardModelFields
    state: CardModelState
    closeTab: (id: string) => void
    reportFailure: (error: Caught) => void
  },
): CardModelTabState {
  const { store, fields, state, closeTab, reportFailure } = deps
  const close = (tab: string) => {
    const path = store.getPath(id)
    store
      .close(id)
      .then((removed) => {
        if (!removed) return
        fields.forget(id)
        state.forget(
          path,
          store.getOpenIds().some((one) => store.getPath(one) === path),
        )
        closeTab(tab)
      })
      .catch(reportFailure)
  }

  const actions = fields.createActions(id)
  return {
    id,
    data: {
      note: computed(() => store.getOpenNote(id)),
      cardModel: computed(() => fields.getCardModel(id)),
      marks: computed(() => fields.getMarks(id)),
      errorMessage: computed(() =>
        state.getErrorMessage(store.getPath(id), store.getOpenNote(id).error),
      ),
    },
    fields: actions.fields,
    templates: actions.templates,
    file: {
      keepMine: () => {
        store.keep(id).catch(reportFailure)
      },
      takeFile: () => {
        store.take(id).catch(reportFailure)
      },
      close,
    },
  }
}

function createCardModelKindDefinition(deps: {
  store: ReturnType<typeof openNotes>
  state: CardModelState
  cards: Cards
  createCardModelTabState: (id: string) => CardModelTabState
  component: Component
}): { kind: TabKindRegistration<CardModelTabState> } {
  const { store, state, cards } = deps
  const getTitle = (path: string): string => state.getTitle(path) || getFileName(path)
  const kind: TabKindRegistration<CardModelTabState> = {
    kind: CARD_MODEL,
    component: deps.component,
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
    getMark: (one) => getMarkOf(one.data.note.value.state),
    getTarget: (one) => ({
      path: store.getPath(one.id),
      title: getTitle(store.getPath(one.id)),
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
    createFile: (folder, name) => cards.createCardModel(name, folder, [words.newField]),
  }
  return { kind }
}

export function createCardModelTabs(cards: Cards, options: CardModelTabsOptions) {
  // Messages are dropped when the caller supplies no writer.
  const { buffers, closeTab, writeMessage = () => undefined, component } = options
  const reportFailure = (error: Caught): void => writeMessage(formatErrorMessage(error), 'error')
  const state = createCardModelState()
  const wire = createCardModelWire(cards, state, writeMessage)
  const store = openNotes({ read: wire.read, write: wire.write })
  buffers.track(store)
  const fields = createCardModelFields(
    (id) => store.getOpenNote(id).body,
    (id, body) => store.setBody(id, body),
    (id) => state.getProblems(store.getPath(id)),
    // The vault renames the field in the file on disk, so the edits the tab holds are written first.
    (id, field, name) => {
      store
        .flushPending(id)
        .then(() =>
          wire.renameField(store.getPath(id), { from: field, to: name }, store.applyPathChanges),
        )
        .catch(reportFailure)
    },
  )

  const createCardModelTabState = (id: string): CardModelTabState =>
    createCardModelTabItem(id, { store, fields, state, closeTab, reportFailure })
  const { kind } = createCardModelKindDefinition({
    store,
    state,
    cards,
    createCardModelTabState,
    component,
  })

  return {
    kind,
    createCardModelTabState,
    applyPathChanges: (paths: readonly string[], renames: readonly PathRename[] = []) => {
      state.movePaths(renames)
      store.applyPathChanges(paths, renames)
    },
    getTitle: (path: string): string => state.getTitle(path) || getFileName(path),
    store,
  }
}
