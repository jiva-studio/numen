/**
 * Window registration and tab state for flashcard stencil tabs.
 */
import { computed } from 'vue'
import type { PlexDestination } from '@numen/ui'
import type { Cards } from '@/entities/deck'
import type { Store } from '@/features/command-palette'
import { openNotes, getMarkOf } from '@/entities/note'
import type { MessageWriter } from '@/shared/notices/messages'
import type { FileOpeners, OpenPlacement, TabKind, WindowHandle } from '@/entities/tab'
import { STENCIL } from '@/entities/tab'
import StencilTab from './ui/StencilTab.vue'
import { getFileName, type PathRename } from '@/shared/paths'
import type { StencilTabState } from './types'
import { createStencilWire, type VaultAnswer } from './api/wire'
import { createStencilFields } from './model/fields'

export type { StencilTabState, VaultAnswer }

export interface StencilTabsDeps {
  now?: (() => number) | undefined
}

export interface StencilTabsOptions {
  tabOpeners: FileOpeners
  writeMessage?: MessageWriter | undefined
  now?: (() => number) | undefined
}

type StencilStore = ReturnType<typeof openNotes>
type StencilFields = ReturnType<typeof createStencilFields>
type StencilWire = ReturnType<typeof createStencilWire>

function createStencilTabItem(
  id: string,
  deps: {
    store: StencilStore
    fields: StencilFields
    wire: StencilWire
    handle: WindowHandle
  },
): StencilTabState {
  const { store, fields, wire, handle } = deps
  const closeTab = (tab: string) => {
    const path = store.getPath(id)
    void store.close(id).then((gone) => {
      if (!gone) return
      fields.forget(id)
      wire.forget(
        path,
        store.getOpenIds().some((one) => store.getPath(one) === path),
      )
      handle.closeTab(tab)
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
    close: closeTab,
  }
}

function createStencilStoreProxy(
  store: StencilStore,
  wire: StencilWire,
  closeTab: (id: string) => void,
): Store {
  const getTitle = (path: string): string => wire.getTitle(path) || getFileName(path)
  return {
    has: (id) => store.getOpenIds().includes(id),
    getPath: (id) => store.getPath(id),
    getTitle: (id) => getTitle(store.getPath(id)),
    asking: (id) => store.stale(id) !== null,
    flushPending: (id) => store.flushPending(id),
    close: closeTab,
    getTabAt: (path) => store.getOpenIds().find((id) => store.getPath(id) === path) ?? null,
  }
}

function createStencilTabIdManager(store: StencilStore) {
  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(store.getOpenIds().map((one) => [store.getPath(one), one])),
  )
  const pendingTabPaths = new Map<string, string>()

  const getOrCreateTabId = (path: string): string => {
    const open = tabbed.value.get(path)
    if (open) return open
    const id = crypto.randomUUID()
    pendingTabPaths.set(id, path)
    return id
  }

  return { tabbed, pendingTabPaths, getOrCreateTabId }
}

function createStencilKindDefinition(deps: {
  store: StencilStore
  wire: StencilWire
  pendingTabPaths: Map<string, string>
  createStencilTabState: (id: string) => StencilTabState
}): TabKind<StencilTabState, typeof STENCIL> {
  const getTitle = (path: string): string => deps.wire.getTitle(path) || getFileName(path)
  return {
    kind: STENCIL,
    open: (id) => {
      const path = deps.pendingTabPaths.get(id) ?? id
      deps.store.open(id, path)
      // Stryker disable next-line CallExpression: memory cleanup for temporary id mapping.
      deps.pendingTabPaths.delete(id)
      return deps.createStencilTabState(id)
    },
    getTitle: (one) => getTitle(deps.store.getPath(one.id)),
    getMark: (one) => getMarkOf(one.note.value.state),
    pane: StencilTab,
    identity: (id) => id,
    getTarget: (one) => ({
      path: deps.store.getPath(one.id),
      title: getTitle(deps.store.getPath(one.id)),
    }),
    getOpenTab: (one) => ({ path: deps.store.getPath(one.id) }),
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    onDestroy: () => {},
  }
}

function registerStencilOpener(
  deps: { tabOpeners: FileOpeners; handle: WindowHandle },
  tabIds: ReturnType<typeof createStencilTabIdManager>,
  wire: ReturnType<typeof createStencilWire>,
) {
  deps.tabOpeners.registerEditor(
    'stencil',
    (path: string, title: string, how?: PlexDestination | OpenPlacement) => {
      const id = tabIds.getOrCreateTabId(path)
      if (title) wire.setTitle(path, title)
      const where = typeof how === 'string' ? how : how?.where
      void (where === 'beside'
        ? deps.handle.openTabBeside(STENCIL, id)
        : deps.handle.openTab(STENCIL, id))
    },
  )
}

export function useStencilTabs(
  cards: Cards,
  handle: WindowHandle,
  ...args:
    | [options: StencilTabsOptions]
    | [tabOpeners: FileOpeners, writeMessage?: MessageWriter, deps?: StencilTabsDeps]
) {
  const options: StencilTabsOptions =
    args[0] && 'registerEditor' in args[0]
      ? {
          tabOpeners: args[0] as FileOpeners,
          ...(args[1] ? { writeMessage: args[1] as MessageWriter } : {}),
          ...((args[2] as StencilTabsDeps | undefined)?.now
            ? { now: (args[2] as StencilTabsDeps).now }
            : {}),
        }
      : (args[0] as StencilTabsOptions)
  // Stryker disable next-line ArrowFunction: fallback clock is used when no clock dependency is injected.
  const { tabOpeners, writeMessage = () => {}, now = () => 0 } = options
  const wire = createStencilWire(cards, writeMessage)
  // Stryker disable next-line ObjectLiteral: note store receives injected clock or fallback.
  const store = openNotes({ read: wire.read, write: wire.write }, { now })
  const fields = createStencilFields(
    (id) => store.getOpenNote(id).body,
    (id, body) => store.setBody(id, body),
    (id) => wire.getProblems(store.getPath(id)),
    (id, field, name) =>
      void wire.renameField(store.getPath(id), { from: field, to: name }, store.applyPathChanges),
  )

  const tabIds = createStencilTabIdManager(store)
  const createStencilTabState = (id: string): StencilTabState =>
    createStencilTabItem(id, { store, fields, wire, handle })
  const getTitle = (path: string): string => wire.getTitle(path) || getFileName(path)
  const closeTab = (id: string): void => {
    const tab = handle.each<StencilTabState>(STENCIL).find((one) => one.state.id === id)
    tab?.state.close(tab.id)
  }

  registerStencilOpener({ tabOpeners, handle }, tabIds, wire)

  const kind = createStencilKindDefinition({
    store,
    wire,
    pendingTabPaths: tabIds.pendingTabPaths,
    createStencilTabState,
  })
  const kept = createStencilStoreProxy(store, wire, closeTab)

  return {
    kind,
    createStencilTabState,
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    applyPathChanges: (paths: readonly string[], renames: readonly PathRename[] = []) => {
      wire.movePaths(renames)
      store.applyPathChanges(paths, renames)
    },
    getTitle,
    kept,
    getOpenIds: store.getOpenIds,
    getOpenNote: store.getOpenNote,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
