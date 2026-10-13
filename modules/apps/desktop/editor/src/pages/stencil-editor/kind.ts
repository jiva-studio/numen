/**
 * Window registration and tab state for flashcard stencil tabs.
 */
import { computed } from 'vue'
import type { PlexDestination } from '@numen/ui'
import type { Cards } from '@/entities/deck'
import type { Store } from '@/features/command-palette'
import { openNotes, getMarkOf } from '@/entities/note'
import type { MessageWriter } from '@/shared/notices/messages'
import type { TabKind, WindowHandle } from '@/entities/tab'
import type { FileOpeners } from '@/entities/tab'
import { STENCIL } from '@/entities/tab'
import StencilTab from './ui/StencilTab.vue'
import { getFileName, type PathRename } from '@/shared/paths'
import type { StencilTabState } from './types'
import { createStencilWire, type VaultAnswer } from './api/wire'
import { createStencilFields } from './model/fields'

export type { StencilTabState, VaultAnswer }

export interface StencilTabsDeps {
  now?: () => number
}

export function useStencilTabs(
  cards: Cards,
  handle: WindowHandle,
  tabOpeners: FileOpeners,
  writeMessage: MessageWriter = () => {},
  deps?: StencilTabsDeps,
) {
  const wire = createStencilWire(cards, writeMessage)
  const now = deps?.now ?? (() => 0)

  const store = openNotes(
    {
      read: wire.read,
      write: wire.write,
    },
    { now },
  )

  const fields = createStencilFields(
    (id) => store.getOpenNote(id).body,
    (id, body) => store.setBody(id, body),
    (id) => wire.getProblems(store.getPath(id)),
    (id, field, name) =>
      void wire.renameField(store.getPath(id), { from: field, to: name }, store.applyPathChanges),
  )

  const createStencilTabState = (id: string): StencilTabState => {
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

  /** What a stencil tab is called: the title the file carries, or the file itself. */
  const getTitle = (path: string): string => wire.getTitle(path) || getFileName(path)

  /** The tab holding a stencil lets go of it, wherever the window draws it. */
  const closeTab = (id: string): void => {
    const tab = handle.each<StencilTabState>(STENCIL).find((one) => one.state.id === id)
    tab?.state.close(tab.id)
  }

  /** The stencils, as a command reaches the ones the window has open. */
  const kept: Store = {
    has: (id) => store.getOpenIds().includes(id),
    getPath: (id) => store.getPath(id),
    getTitle: (id) => getTitle(store.getPath(id)),
    asking: (id) => store.stale(id) !== null,
    flushPending: (id) => store.flushPending(id),
    close: closeTab,
    getTabAt: (path) => store.getOpenIds().find((id) => store.getPath(id) === path) ?? null,
  }

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

  const kind: TabKind<StencilTabState, typeof STENCIL> = {
    kind: STENCIL,
    open: (id) => {
      const path = pendingTabPaths.get(id) ?? id
      store.open(id, path)
      pendingTabPaths.delete(id)
      return createStencilTabState(id)
    },
    getTitle: (one) => getTitle(store.getPath(one.id)),
    getMark: (one) => getMarkOf(one.note.value.state),
    pane: StencilTab,
    identity: (id) => id,
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
  }

  const openStencil = (path: string, title: string, how: PlexDestination): void => {
    const id = getOrCreateTabId(path)
    if (title) wire.setTitle(path, title)
    void (how === 'beside' ? handle.openTabBeside(STENCIL, id) : handle.openTab(STENCIL, id))
  }

  tabOpeners.registerEditor('stencil', openStencil)

  const applyPathChanges = (
    paths: readonly string[],
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    renames: readonly PathRename[] = [],
  ): void => {
    wire.movePaths(renames)
    store.applyPathChanges(paths, renames)
  }

  return {
    kind,
    createStencilTabState,
    applyPathChanges,
    getTitle,
    kept,
    getOpenIds: store.getOpenIds,
    getOpenNote: store.getOpenNote,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
