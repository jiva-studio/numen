/**
 * Window registration and navigation for plex graph tabs.
 */
import type { TabKind, WindowHandle } from '@/entities/tab'
import { PLEX } from '@/entities/tab'
import type { PathRename } from '@/shared/paths'
import { usePlexTab } from './model/usePlexTab'
import type { PlexView } from './model/usePlexView'
import { WORDS as words } from './words'
import type { PlexTabDeps, PlexTabState } from './types'
import PlexTab from './ui/PlexTab.vue'

/** What a plex tab is called: the note it stands on. */
const titleOf = (note: string): string => note || words.plex

/**
 * Manages window-level plex tab operations and navigation.
 */
export function plexKind(handle: WindowHandle, createView: () => PlexView, deps: PlexTabDeps) {
  const all = () => handle.each<PlexTabState>(PLEX)
  const front = (): PlexTabState | null => handle.last<PlexTabState>(PLEX)?.state ?? null

  const kind: TabKind<PlexTabState, typeof PLEX> = {
    kind: PLEX,
    open: (at) => {
      const state = usePlexTab(createView(), deps)
      const from = at || getCurrentPath() || deps.openingPath.value
      if (from) void state.view.go(from)
      return state
    },
    getTitle: (state) => titleOf(state.view.neighbourhood.value?.focus.title ?? ''),
    pane: PlexTab,
    onClose: (state) => {
      state.view.close()
      return true
    },
    over: (state) => {
      const path = state.view.here.value
      return { path, title: (path && state.getName(path)) || path }
    },
    getOpenTab: (state) => ({ path: state.view.here.value }),
  }

  const getCurrentPath = (): string => front()?.view.here.value ?? ''

  const getName = (path: string): string => (path ? (front()?.getName(path) ?? '') : '')

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
      all()
        .filter(({ state }) => state.view.here.value === from)
        .map(({ state }) => state.view.go(to)),
    )
  }

  const refresh = async (renames: readonly PathRename[] = []) => {
    if (renames.length) for (const { state } of all()) state.followMoves(renames)
    if (all().some(({ state }) => !state.view.here.value)) {
      try {
        await deps.readOpeningPath()
      } catch {
        // The next change asks again.
      }
    }
    await Promise.all(
      all().map(async ({ state }) => {
        const path = state.view.here.value || deps.openingPath.value
        if (path) await state.view.go(path)
        await state.readParts()
      }),
    )
  }

  return { kind, looking: getCurrentPath, getName, travel, leavePath, refresh }
}
