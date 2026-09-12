/**
 * Window registration and tab state for flashcard stencil tabs.
 */
import { computed } from 'vue'
import type { PlexShowing } from '@numen/ui'
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
import { createStencilFields } from './stencilTabs.fields'

export type { StencilTabState, VaultAnswer }

export function useStencilTabs(
  cards: Cards,
  handle: WindowHandle,
  puts: FileOpeners,
  says: MessageWriter = () => {},
) {
  const wire = createStencilWire(cards, says)

  const store = openNotes({
    read: wire.read,
    write: wire.write,
  })

  const fields = createStencilFields(
    (id) => store.shown(id).body,
    (id, body) => store.typed(id, body),
    (id) => wire.getProblems(store.where(id)),
    (id, field, name) => void wire.renameField(store.where(id), field, name, store.changed),
  )

  const held = (id: string): StencilTabState => {
    const closeTab = (tab: string) => {
      const path = store.where(id)
      void store.shut(id).then((gone) => {
        if (!gone) return
        fields.forget(id)
        wire.forget(path, store.all().some((one) => store.where(one) === path))
        handle.closes(tab)
      })
    }

    return {
      id,
      shown: computed(() => store.shown(id)),
      stencil: computed(() => fields.getStencil(id)),
      marks: computed(() => fields.getMarks(id)),
      errorMessage: computed(() => wire.getErrorMessage(store.where(id), store.shown(id).error)),
      ...fields.actionsFor(id),
      keepMine: () => store.keep(id),
      takeFile: () => store.take(id),
      close: closeTab,
    }
  }

  /** What a stencil tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string => wire.getTitle(path) || fileOf(path)

  /** The tab holding a stencil lets go of it, wherever the window draws it. */
  const shuts = (id: string): void => {
    const tab = handle.each<StencilTabState>(STENCIL).find((one) => one.state.id === id)
    tab?.state.close(tab.id)
  }

  /** The stencils, as a command reaches the ones the window has open. */
  const kept: Store = {
    has: (id) => store.all().includes(id),
    where: (id) => store.where(id),
    called: (id) => called(store.where(id)),
    asking: (id) => store.stale(id) !== null,
    settles: (id) => store.settles(id),
    shuts,
    holding: (path) => store.all().find((id) => store.where(id) === path) ?? null,
  }

  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(store.all().map((one) => [store.where(one), one])),
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
  const mints = getOrCreateTabId

  const kind: TabKind<StencilTabState, typeof STENCIL> = {
    kind: STENCIL,
    opens: (id) => {
      const path = pendingTabPaths.get(id) ?? id
      store.open(id, path)
      pendingTabIds.delete(path)
      pendingTabPaths.delete(id)
      return held(id)
    },
    called: (one) => called(store.where(one.id)),
    marked: (one) => markOf(one.shown.value.state),
    draws: StencilTab,
    identity: (id) => id,
    shuts: (one, id) => {
      one.close(id)
      return false
    },
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    gone: () => {},
  }

  const shows = (path: string, title = '', showing: PlexShowing = 'here'): void => {
    const id = mints(path)
    if (title) wire.setTitle(path, title)
    void (showing === 'beside' ? handle.beside(STENCIL, id) : handle.opens(STENCIL, id))
  }

  puts.holds('stencil', shows)

  const changed = (paths: readonly string[], renamed: readonly PathRename[] = []): void => {
    wire.movePaths(renamed)
    store.changed(paths, renamed)
  }

  return {
    kind,
    held,
    changed,
    called,
    kept,
    all: store.all,
    shown: store.shown,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
