/**
 * Window registration and tab state for note tabs.
 */
import { computed, type ComputedRef } from 'vue'
import type { PlexDestination } from '@numen/ui'
import type { Store } from '@/features/command-palette'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { NOTE } from '@/entities/tab'
import type { createNoteChanges } from './model/changes'
import { getMarkOf, type openNotes } from '@/entities/note'
import { createNoteKeyboard, ITSELF } from './model/keyboard'
import { createNoteTitles, type NoteTitlesDeps } from './model/titles'
import NoteTab from './ui/NoteTab.vue'
import type { FileOpeners } from '@/entities/tab'
import type { NoteTabDeps, NoteTabState } from './types'
import { createNoteTab } from './model/open'

export type { NoteTabDeps, NoteTabState, NoteTitlesDeps }

/** The notes of the whole window, read and written by one store. */
type Notes = ReturnType<typeof openNotes>
/** What is being typed into each note now, as the editor draws it. */
type NoteChanges = ReturnType<typeof createNoteChanges>

export interface NoteTabOptions {
  changes: NoteChanges
  handle: WindowHandle
  tabOpeners: FileOpeners
}

function createNoteTabIdManager(notes: Notes) {
  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(notes.getOpenIds().map((id) => [notes.getPath(id), id])),
  )
  const pendingIds = new Map<string, string>()
  const pendingPaths = new Map<string, string>()

  const getOrCreateTabId = (path: string): string => {
    const open = tabbed.value.get(path) ?? pendingIds.get(path)
    if (open) return open
    const id = crypto.randomUUID()
    pendingIds.set(path, id)
    pendingPaths.set(id, path)
    return id
  }

  const getTabId = (path: string): string => tabbed.value.get(path) ?? pendingIds.get(path) ?? path

  return { tabbed, pendingIds, pendingPaths, getOrCreateTabId, getTabId }
}

function createNoteKindDefinition(deps: {
  openTab: (id: string, line?: number) => NoteTabState
  names: ReturnType<typeof createNoteTitles>
  notes: Notes
}): TabKind<NoteTabState, typeof NOTE> {
  const getPath = (state: NoteTabState): string =>
    deps.notes.has(state.id) ? deps.notes.getPath(state.id) : ''

  return {
    kind: NOTE,
    open: (id) => deps.openTab(id),
    getTitle: (state) => deps.names.getTitle(state.id),
    getMark: (state) => getMarkOf(state.note.value.state),
    pane: NoteTab,
    identity: (id) => id,
    onShow: (state) => state.measure(),
    getTarget: (state) => {
      const path = getPath(state)
      return { path, title: path ? deps.names.getTitle(state.id) : '' }
    },
    getOpenTab: (state) => ({ path: getPath(state) }),
    onClose: (state, id) => {
      state.close(id)
      return false
    },
    onDestroy: () => {},
  }
}

function createNoteStoreProxy(
  notes: Notes,
  names: ReturnType<typeof createNoteTitles>,
  ...[closeTab, tabbed]: [
    closeTab: (id: string) => void,
    tabbed: ComputedRef<ReadonlyMap<string, string>>,
  ]
): Store {
  return {
    has: (id) => notes.has(id),
    getPath: (id) => notes.getPath(id),
    getTitle: (id) => names.getTitle(id),
    asking: (id) => notes.stale(id) !== null,
    flushPending: (id) => notes.flushPending(id),
    close: closeTab,
    getTabAt: (path) => tabbed.value.get(path) ?? null,
  }
}

function createNoteTabLoader(
  notes: Notes,
  names: ReturnType<typeof createNoteTitles>,
  changes: NoteChanges,
) {
  const loadInTab = async (id: string, path: string, title = '') => {
    const currentPath = notes.getPath(id)
    if (currentPath === path) {
      if (title) names.setTitle(id, title)
      return
    }
    changes.closeNote(currentPath)
    await notes.flushPending(id)
    notes.reload(id, path)
    if (title) {
      names.setTitle(id, title)
    } else {
      names.forgetTab(id)
    }
  }
  return { loadInTab }
}

export function useNoteTab(
  vault: NoteTabDeps,
  notes: Notes,
  ...args:
    | [options: NoteTabOptions]
    | [changes: NoteChanges, handle: WindowHandle, tabOpeners: FileOpeners]
) {
  const { changes, handle, tabOpeners } =
    args.length === 1 ? args[0] : { changes: args[0], handle: args[1], tabOpeners: args[2] }
  const names = createNoteTitles(vault, notes)
  const keyboard = createNoteKeyboard()
  const tabIds = createNoteTabIdManager(notes)

  const openTab = (id: string, line = ITSELF) => {
    const path = tabIds.pendingPaths.get(id) ?? id
    notes.open(id, path)
    tabIds.pendingIds.delete(path)
    tabIds.pendingPaths.delete(id)
    keyboard.requestFocus(id, line)
    return createNoteTabState(id)
  }

  const openNote = (path: string, title: string, how: PlexDestination) => {
    const id = tabIds.getOrCreateTabId(path)
    if (title) names.setTitle(id, title)
    void (how === 'beside' ? handle.openTabBeside(NOTE, id) : handle.openTab(NOTE, id))
  }

  const focusLine = (path: string, line?: number) =>
    keyboard.requestFocus(tabIds.getTabId(path), line)

  tabOpeners.registerEditor('note', (path, title, opts) => {
    const how = typeof opts === 'string' ? opts : (opts?.where ?? 'here')
    const line = typeof opts === 'object' ? opts?.line : undefined
    openNote(path, title, how)
    focusLine(path, line)
  })

  const closeTab = (id: string) => {
    const tab = handle.each<NoteTabState>(NOTE).find((one) => one.state.id === id)
    tab?.state.close(tab.id)
  }

  const createNoteTabState = (id: string): NoteTabState =>
    createNoteTab(id, { notes, changes, keyboard, vault, names, handle, tabOpeners })

  const kind = createNoteKindDefinition({ openTab, names, notes })
  const kept = createNoteStoreProxy(notes, names, closeTab, tabIds.tabbed)
  const { loadInTab } = createNoteTabLoader(notes, names, changes)

  return {
    kind,
    kept,
    createNoteTabState,
    openTab,
    loadInTab,
    titles: names.titles,
    measureAll: keyboard.measureAll,
    setTitle: (path: string, title: string) => names.setTitle(tabIds.getOrCreateTabId(path), title),
    getTitle: (path: string) => names.getTitle(tabIds.getTabId(path)),
    focusLine,
    closeTab,
  }
}
