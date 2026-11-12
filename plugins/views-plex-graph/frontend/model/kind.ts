/**
 * The plex tab kind: a graph of the vault around one note, with one tab per
 * graph.
 *
 * The kind keeps its own tabs ordered by last use, so a new plex opens beside
 * the most recent one, and a change to the vault updates every plex.
 */
import { watch, type Component } from 'vue'
import { Waypoints } from '@lucide/vue'
import {
  isNotePath,
  type JsonValue,
  type TabKindRegistration,
  type VaultChangeEvent,
} from '@numen/plugin-sdk'
import { createPlexTab } from './usePlexTab'
import { createPlexTabDeps, type PlexHost } from './tabDeps'
import { WORDS as words } from '../words'
import type { PlexTabState, PlexView } from '../types'

/** The word plex tabs are filed under. */
export const PLEX = 'plex'

/** What a plex tab saves across restarts: the note in focus. */
type PlexSavedState = {
  readonly focus: string
}

const readSavedFocus = (saved: JsonValue | undefined): string => {
  const focus = (saved as Partial<PlexSavedState> | null | undefined)?.focus
  return typeof focus === 'string' ? focus : ''
}

const getPlexTitle = (note: string): string => note || words.plex

function createPlexTabs() {
  const tabs = new Map<string, PlexTabState>()
  const getLastTab = (): PlexTabState | null => [...tabs.values()].at(-1) ?? null
  return {
    add: (id: string, state: PlexTabState) => {
      tabs.set(id, state)
    },
    remove: (id: string) => {
      tabs.delete(id)
    },
    show: (id: string) => {
      const state = tabs.get(id)
      if (!state) return
      tabs.delete(id)
      tabs.set(id, state)
    },
    find: (id: string): PlexTabState | undefined => tabs.get(id),
    getAll: (): readonly PlexTabState[] => [...tabs.values()],
    getCurrentPath: (): string => getLastTab()?.canvas.view.here.value ?? '',
  }
}

async function readOpeningPath(host: PlexHost): Promise<boolean> {
  try {
    await host.workspace.window.readOpeningPath()
    return true
  } catch {
    return false
  }
}

async function refreshTabs(
  host: PlexHost,
  tabs: readonly PlexTabState[],
  event: VaultChangeEvent,
): Promise<void> {
  for (const state of tabs) state.nodes.followMoves(event.renamed ?? [])
  if (tabs.some((state) => !state.canvas.view.here.value)) {
    await readOpeningPath(host)
  }
  await Promise.all(
    tabs.map(async (state) => {
      const path =
        state.canvas.view.nextPath.value ||
        state.canvas.view.here.value ||
        host.workspace.window.openingPath.value
      if (path) await state.canvas.view.go(path)
      await state.headings.readHeadings()
    }),
  )
}

/**
 * The plex kind as the window keeps it, and what a change to the vault does to
 * its tabs. Each tab draws a view of its own, made by `createView` and drawn by `component`.
 */
export function createPlexKind(host: PlexHost, createView: () => PlexView, component: Component) {
  const tabs = createPlexTabs()

  const kind: TabKindRegistration<PlexTabState> = {
    kind: PLEX,
    component,
    title: words.plex,
    icon: Waypoints,
    isTool: true,
    placement: { pane: 'main', order: 0 },
    open: (targetPath, id, saved) => {
      const from =
        readSavedFocus(saved) ||
        targetPath ||
        tabs.getCurrentPath() ||
        host.workspace.window.openingPath.value
      const state = createPlexTab(createView(), createPlexTabDeps(host, id))
      tabs.add(id, state)
      if (from) state.canvas.view.go(from).catch(state.reportFailure)
      watch(state.canvas.view.here, (path) => {
        if (path) host.workspace.files.syncPath(id, path, state.nodes.getName(path))
      })
      return state
    },
    getTitle: (state) => getPlexTitle(state.canvas.view.neighborhood.value?.focus.title ?? ''),
    onShow: (_state, id) => tabs.show(id),
    onClose: (state, id) => {
      state.canvas.view.close()
      tabs.remove(id)
      return true
    },
    getTarget: (state) => {
      const path = state.canvas.view.here.value
      return { path, title: (path && state.nodes.getName(path)) || path }
    },
    getOpenTab: (state) => ({ path: state.canvas.view.here.value }),
    getState: (state): PlexSavedState => ({ focus: state.canvas.view.here.value }),
    navigate: async (state, path) => {
      if (isNotePath(path) && state.canvas.view.here.value !== path)
        await state.canvas.view.go(path)
    },
  }

  const refresh = (event: VaultChangeEvent) => refreshTabs(host, tabs.getAll(), event)

  return { kind, refresh }
}
