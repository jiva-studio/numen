/**
 * The stencil tab kind and the state of its tabs: a stencil edited as its fields and faces.
 */
import { computed } from 'vue'
import { LayoutTemplate } from '@lucide/vue'
import {
  getFileName,
  type BufferHost,
  type MessageWriter,
  type PathRename,
  type TabKindRegistration,
} from '@numen/plugin-sdk'
import { getMarkOf, openNotes } from '../buffer'
import { WORDS as words, type Cards } from '../cards'
import StencilTab from './ui/StencilTab.vue'
import type { StencilTabState } from './types'
import { createStencilWire, type VaultAnswer } from './api/wire'
import { createStencilFields } from './model/fields'

export type { StencilTabState, VaultAnswer }

/** The word stencil tabs are filed under. */
export const STENCIL = 'stencil'

/** What the stencil tabs of a window are made from. */
export interface StencilTabsOptions {
  readonly buffers: BufferHost
  /** The window lets go of a tab that took its own close. */
  readonly closeTab: (id: string) => void
  readonly writeMessage?: MessageWriter | undefined
}

type StencilFields = ReturnType<typeof createStencilFields>
type StencilWire = ReturnType<typeof createStencilWire>

function createStencilTabItem(
  id: string,
  deps: {
    store: ReturnType<typeof openNotes>
    fields: StencilFields
    wire: StencilWire
    closeTab: (id: string) => void
  },
): StencilTabState {
  const { store, fields, wire, closeTab } = deps
  const close = (tab: string) => {
    const path = store.getPath(id)
    void store.close(id).then((removed) => {
      if (!removed) return
      fields.forget(id)
      wire.forget(
        path,
        store.getOpenIds().some((one) => store.getPath(one) === path),
      )
      closeTab(tab)
    })
  }

  return {
    id,
    note: computed(() => store.getOpenNote(id)),
    stencil: computed(() => fields.getStencil(id)),
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

function createStencilKindDefinition(deps: {
  store: ReturnType<typeof openNotes>
  wire: StencilWire
  cards: Cards
  createStencilTabState: (id: string) => StencilTabState
}): TabKindRegistration<StencilTabState> {
  const { store, wire, cards } = deps
  const getTitle = (path: string): string => wire.getTitle(path) || getFileName(path)
  return {
    kind: STENCIL,
    component: StencilTab,
    icon: LayoutTemplate,
    open: (path, id) => {
      store.open(id, path)
      return deps.createStencilTabState(id)
    },
    // A stencil already open under the path is the tab of that path, wherever the file has since moved.
    identity: (path) => {
      const open = store.getOpenIds().find((id) => store.getPath(id) === path)
      return open === undefined ? path : open.slice(STENCIL.length + 1)
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
    onDestroy: () => {},
    isAsking: (one) => store.stale(one.id) !== null,
    flushPending: (one) => store.flushPending(one.id),
    createFile: (folder, name) => cards.createStencil(name, folder, [words.newField]),
  }
}

export function createStencilTabs(cards: Cards, options: StencilTabsOptions) {
  const { buffers, closeTab, writeMessage = () => {} } = options
  const wire = createStencilWire(cards, writeMessage)
  const store = openNotes({ read: wire.read, write: wire.write })
  buffers.track(store)
  const fields = createStencilFields(
    (id) => store.getOpenNote(id).body,
    (id, body) => store.setBody(id, body),
    (id) => wire.getProblems(store.getPath(id)),
    (id, field, name) =>
      void wire.renameField(store.getPath(id), { from: field, to: name }, store.applyPathChanges),
  )

  const createStencilTabState = (id: string): StencilTabState =>
    createStencilTabItem(id, { store, fields, wire, closeTab })
  const kind = createStencilKindDefinition({ store, wire, cards, createStencilTabState })

  return {
    kind,
    createStencilTabState,
    applyPathChanges: (paths: readonly string[], renames: readonly PathRename[] = []) => {
      wire.movePaths(renames)
      store.applyPathChanges(paths, renames)
    },
    getTitle: (path: string): string => wire.getTitle(path) || getFileName(path),
    store,
  }
}

export const useStencilTabs = createStencilTabs
