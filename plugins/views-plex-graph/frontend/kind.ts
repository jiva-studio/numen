/**
 * The plex tab kind: a graph of the vault around one note, a tab of it per graph.
 *
 * The kind keeps its own tabs in the order the person was last in them, so a
 * new plex opens where the last one stands and a change to the vault reaches
 * every plex.
 */
import { watch } from 'vue'
import { Waypoints } from '@lucide/vue'
import { isNotePath, type TabKindRegistration, type VaultChangeEvent } from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'
import { usePlexTab } from './model/usePlexTab'
import { createPlexTabDeps, type PlexHost } from './model/tabDeps'
import { WORDS as words } from './words'
import type { PlexTabState, PlexView } from './types'

/** The word plex tabs are filed under. */
export const PLEX = 'plex'

/** What a plex tab is called: the note it relies on. */
const getPlexTitle = (note: string): string => note || words.plex

/** Every plex tab, the one the person was last in last. */
function createPlexTabs() {
  const tabs = new Map<string, PlexTabState>()
  const getLast = (): PlexTabState | null => [...tabs.values()].at(-1) ?? null
  return {
    add: (id: string, state: PlexTabState) => void tabs.set(id, state),
    remove: (id: string) => void tabs.delete(id),
    show: (id: string) => {
      const state = tabs.get(id)
      if (!state) return
      tabs.delete(id)
      tabs.set(id, state)
    },
    getAll: (): readonly PlexTabState[] => [...tabs.values()],
    getCurrentPath: (): string => getLast()?.view.here.value ?? '',
  }
}

/** Every plex follows a change to the vault, and relies on the opening note where it stood on none. */
async function refreshTabs(
  host: PlexHost,
  tabs: readonly PlexTabState[],
  event: VaultChangeEvent,
): Promise<void> {
  for (const state of tabs) state.followMoves(event.renamed ?? [])
  if (tabs.some((state) => !state.view.here.value)) {
    try {
      await host.workspace.readOpeningPath()
    } catch {
      // The next change asks again.
    }
  }
  await Promise.all(
    tabs.map(async (state) => {
      const path =
        state.view.missingPath.value || state.view.here.value || host.workspace.openingPath.value
      if (path) await state.view.go(path)
      await state.readHeadings()
    }),
  )
}

/**
 * The plex kind as the window keeps it, and what a change to the vault does to
 * its tabs. Each tab draws a view of its own, made by `createView`.
 */
export function createPlexKind(host: PlexHost, createView: () => PlexView) {
  const tabs = createPlexTabs()

  const kind: TabKindRegistration<PlexTabState> = {
    kind: PLEX,
    component: PlexGraphView,
    title: words.plex,
    icon: Waypoints,
    isTool: true,
    placement: { pane: 'main', order: 0 },
    open: (targetPath, id) => {
      const from = targetPath || tabs.getCurrentPath() || host.workspace.openingPath.value
      const state = usePlexTab(createView(), createPlexTabDeps(host, id))
      tabs.add(id, state)
      if (from) void state.view.go(from)
      watch(state.view.here, (path) => {
        if (path) host.workspace.syncPath(id, path, state.getName(path))
      })
      return state
    },
    getTitle: (state) => getPlexTitle(state.view.neighbourhood.value?.focus.title ?? ''),
    onShow: (_state, id) => tabs.show(id),
    onClose: (state, id) => {
      state.view.close()
      tabs.remove(id)
      return true
    },
    getTarget: (state) => {
      const path = state.view.here.value
      return { path, title: (path && state.getName(path)) || path }
    },
    getOpenTab: (state) => ({ path: state.view.here.value }),
    navigate: async (state, path) => {
      if (isNotePath(path) && state.view.here.value !== path) await state.view.go(path)
    },
  }

  const refresh = (event: VaultChangeEvent) => refreshTabs(host, tabs.getAll(), event)

  return { kind, refresh }
}
