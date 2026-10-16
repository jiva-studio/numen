/**
 * Window registration and navigation for plex graph tabs.
 */
import { watch } from 'vue'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { PLEX } from '@/entities/tab'
import type { PathRename } from '@/shared/paths'
import {
  usePlexTab,
  PlexTab,
  WORDS as words,
  type PlexTabDeps,
  type PlexTabState,
  type PlexView,
} from '@plugins/view-plex-graph/frontend'

/** What a plex tab is called: the note it stands on. */
const getPlexTitle = (note: string): string => note || words.plex

function createPlexTabDefinition(
  createView: () => PlexView,
  deps: PlexTabDeps,
  getCurrentPath: () => string,
): TabKind<PlexTabState, typeof PLEX> {
  return {
    kind: PLEX,
    open: (at) => {
      const current: { state?: PlexTabState } = {}
      const wrappedDeps: PlexTabDeps = {
        ...deps,
        openNote: (path, options) =>
          current.state && deps.openNote(path, { ...options, state: current.state }),
        onSelectPath: (path, title) =>
          current.state && deps.onSelectPath?.(path, title, current.state),
      }
      const tabState = {
        ...usePlexTab(createView(), wrappedDeps),
        pluginId: 'md.numen.view-plex-graph',
      }
      current.state = tabState
      const from = at || getCurrentPath() || deps.openingPath.value
      if (from) void tabState.view.go(from)
      watch(tabState.view.here, (path) => {
        if (path) {
          deps.onSelectPath?.(path, tabState.getName(path), tabState)
        }
      })
      return tabState
    },
    getTitle: (state) => getPlexTitle(state.view.neighbourhood.value?.focus.title ?? ''),
    pane: PlexTab,
    onClose: (state) => {
      state.view.close()
      return true
    },
    getTarget: (state) => {
      const path = state.view.here.value
      return { path, title: (path && state.getName(path)) || path }
    },
    getOpenTab: (state) => ({ path: state.view.here.value }),
  }
}

/**
 * Manages window-level plex tab operations and navigation.
 */
export function createPlexKind(
  handle: WindowHandle,
  createView: () => PlexView,
  deps: PlexTabDeps,
) {
  const getAll = () => handle.each<PlexTabState>(PLEX)
  const front = (): PlexTabState | null => handle.last<PlexTabState>(PLEX)?.state ?? null
  const getCurrentPath = (): string => front()?.view.here.value ?? ''
  const getName = (path: string): string => (path ? (front()?.getName(path) ?? '') : '')

  const kind = createPlexTabDefinition(createView, deps, getCurrentPath)

  const travel = async (path: string) => {
    const one = handle.last<PlexTabState>(PLEX)
    if (!one) {
      await handle.openTab(PLEX, path)
      return
    }
    handle.show(one.id)
    await one.state.view.go(path)
  }

  const leavePath = async (from: string, to: string) => {
    await Promise.all(
      getAll()
        .filter(({ state }) => state.view.here.value === from)
        .map(({ state }) => state.view.go(to)),
    )
  }

  // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
  const refresh = async (renames: readonly PathRename[] = []) => {
    for (const { state } of getAll()) state.followMoves(renames)
    if (getAll().some(({ state }) => !state.view.here.value)) {
      try {
        await deps.readOpeningPath()
      } catch {
        // The next change asks again.
      }
    }
    await Promise.all(
      getAll().map(async ({ state }) => {
        const path = state.view.here.value || deps.openingPath.value
        if (path) await state.view.go(path)
        await state.readHeadings()
      }),
    )
  }

  return { kind, looking: getCurrentPath, getName, travel, leavePath, refresh }
}
