/**
 * Window registration and tab state for flashcard stencil tabs.
 */
import { computed } from 'vue'
import type { PlexDestination } from '@numen/ui'
import type { Cards } from '@/entities/deck'
import type { Store } from '@/features/command-palette'
import { openNotes, markOf } from '@/entities/note'
import type { MessageWriter } from '@/shared/notices/messages'
import type { TabKind, WindowHandle } from '@/entities/tab'
import type { FileOpeners } from '@/entities/tab'
import { STENCIL } from '@/entities/tab'
import StencilTab from './ui/StencilTab.vue'
import { fileOf, type PathRename } from '@/shared/paths'
import type { StencilTabState } from './types'
import { createStencilWire, type VaultAnswer } from './api/wire'
import { createStencilFields } from './model/fields'

export type { StencilTabState, VaultAnswer }

export function useStencilTabs(
  cards: Cards,
  handle: WindowHandle,
  tabOpeners: FileOpeners,
  writeMessage: MessageWriter = () => {},
) {
  const wire = createStencilWire(cards, writeMessage)

  const store = openNotes({
    read: wire.read,
    write: wire.write,
  })

  const fields = createStencilFields(
    (id) => store.getOpenNote(id).body,
    (id, body) => store.setBody(id, body),
    (id) => wire.getProblems(store.getPath(id)),
    (id, field, name) => void wire.renameField(store.getPath(id), field, name, store.changed),
  )

  const createStencilTabState = (id: string): StencilTabState => {
    const closeTab = (tab: string) => {
      const path = store.getPath(id)
      void store.close(id).then((gone) => {
        if (!gone) return
        fields.forget(id)
        wire.forget(path, store.getOpenIds().some((one) => store.getPath(one) === path))
        handle.closeTab(tab)
      })
    }

    return {
      id,
      note: computed(() => store.getOpenNote(id)),
      stencil: computed(() => fields.getStencil(id)),
      marks: computed(() => fields.getMarks(id)),
      errorMessage: computed(() => wire.getErrorMessage(store.getPath(id), store.getOpenNote(id).error)),
      ...fields.actionsFor(id),
      keepMine: () => store.keep(id),
      takeFile: () => store.take(id),
      close: closeTab,
    }
  }

  /** What a stencil tab is called: the title the file carries, or the file itself. */
  const getTitle = (path: string): string => wire.getTitle(path) || fileOf(path)

  /** The tab holding a stencil lets go of it, wherever the window draws it. */
  const closeTab = (id: string): void => {
    const tab = handle.each<StencilTabState>(STENCIL).find((one) => one.state.id === id)
    tab?.state.close(tab.id)
  }

  /** The stencils, as a command reaches the ones the window has open. */
  const kept: Store = {
    has: (id) => store.getOpenIds().includes(id),
    where: (id) => store.getPath(id),
    getTitle: (id) => getTitle(store.getPath(id)),
    asking: (id) => store.stale(id) !== null,
    settle: (id) => store.settle(id),
    close: closeTab,
    holding: (path) => store.getOpenIds().find((id) => store.getPath(id) === path) ?? null,
  }

  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(store.getOpenIds().map((one) => [store.getPath(one), one])),
  )

  const pendingTabIds = new Map<string, string>()
  const pendingTabPaths = new Map<string, string>()

  const getOrCreateTabId = (path: string): string => {
    const open = tabbed.value.get(path) ?? pendingTabIds.get(path)
    if (open) return open
    const id = crypto.randomUUID()
    pendingTabIds.set(path, id)
    pendingTabPaths.set(id, path)
    return id
  }

  const kind: TabKind<StencilTabState, typeof STENCIL> = {
    kind: STENCIL,
    open: (id) => {
      const path = pendingTabPaths.get(id) ?? id
      store.open(id, path)
      pendingTabIds.delete(path)
      pendingTabPaths.delete(id)
      return createStencilTabState(id)
    },
    getTitle: (one) => getTitle(store.getPath(one.id)),
    getMark: (one) => markOf(one.note.value.state),
    pane: StencilTab,
    identity: (id) => id,
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    onDestroy: () => {},
  }

  const openStencil = (path: string, title = '', how: PlexDestination = 'here'): void => {
    const id = getOrCreateTabId(path)
    if (title) wire.setTitle(path, title)
    void (how === 'beside' ? handle.openTabBeside(STENCIL, id) : handle.openTab(STENCIL, id))
  }

  tabOpeners.registerEditor('stencil', openStencil)

  const applyPathChanges = (paths: readonly string[], renames: readonly PathRename[] = []): void => {
    wire.movePaths(renames)
    store.changed(paths, renames)
  }

  return {
    kind,
    createStencilTabState,
    changed: applyPathChanges,
    getTitle,
    kept,
    getOpenIds: store.getOpenIds,
    getOpenNote: store.getOpenNote,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
