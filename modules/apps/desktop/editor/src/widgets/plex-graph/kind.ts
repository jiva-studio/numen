/**
 * Window registration and tab state for plex graph tabs.
 */
import type { PathRename } from '../../shared/core'
import type { TabKind, WindowHandle } from '../../shared/tabs/windowTabs'
import { PLEX } from '../../shared/tabs/workspace'
import PlexTab from './PlexTab.vue'
import { WORDS as words } from './words'
import type { MenuRequest, PlexEditor, PlexTabDeps, PlexTabState } from './types'
import type { PlexView } from './view'
import { usePlexTab } from './open'

export type { MenuRequest, PlexEditor, PlexTabDeps, PlexTabState }
export { usePlexTab }

/** What a plex tab is called: the note it stands on. */
const titleOf = (note: string): string => note || words.plex

/**
 * Manages window-level plex tab operations and navigation.
 */
export function plexKind(handle: WindowHandle, makes: () => PlexView, deps: PlexTabDeps) {
  /** Every plex the window holds, and the one the person was last in. */
  const all = () => handle.each<PlexTabState>(PLEX)
  const front = (): PlexTabState | null => handle.last<PlexTabState>(PLEX)?.state ?? null

  const kind: TabKind<PlexTabState, typeof PLEX> = {
    kind: PLEX,
    opens: (at) => {
      const state = usePlexTab(makes(), deps)
      const from = at || looking() || deps.opening.value
      if (from) void state.view.go(from)
      return state
    },
    called: (state) => titleOf(state.view.neighbourhood.value?.focus.title ?? ''),
    getTitle: (state) => titleOf(state.view.neighbourhood.value?.focus.title ?? ''),
    draws: PlexTab,
    shuts: (state) => {
      state.view.close()
      return true
    },
    onClose: (state) => {
      state.view.close()
      return true
    },
    over: (state) => {
      const path = state.view.here.value
      return { path, title: (path && state.nameOf(path)) || path }
    },
    attends: (state) => ({ path: state.view.here.value }),
    getAttention: (state) => ({ path: state.view.here.value }),
  }

  /** The note the person is looking at, which is what a question is about. */
  const looking = (): string => front()?.view.here.value ?? ''

  /** What the plex in front calls a note, and nothing where it names none. */
  const names = (path: string): string => (path ? (front()?.nameOf(path) ?? '') : '')

  /**
   * A note put in front of the person: the plex they are looking at travels
   * there and comes to the front, and a window holding no plex at all opens one
   * on it.
   */
  const travel = async (path: string) => {
    const one = handle.last<PlexTabState>(PLEX)
    if (!one) {
      await handle.opens(PLEX, path)
      return
    }
    handle.shows(one.id)
    await one.state.view.go(path)
  }

  /**
   * Every plex standing on a note travels to another one. A plex standing
   * anywhere else stays where it is.
   */
  const leaves = async (from: string, to: string) => {
    await Promise.all(
      all()
        .filter(({ state }) => state.view.here.value === from)
        .map(({ state }) => state.view.go(to)),
    )
  }

  /**
   * Every plex asks for its picture again, following whatever moved: a plex
   * standing on a note that was renamed stands on where it went. One standing
   * nowhere is given the note the vault opens with, which is asked for once for
   * all of them and only while one of them has nowhere to stand.
   */
  const again = async (renamed: readonly PathRename[] = []) => {
    if (renamed.length) for (const { state } of all()) state.follows(renamed)
    if (all().some(({ state }) => !state.view.here.value)) {
      try {
        await deps.first()
      } catch {
        // The next change asks again.
      }
    }
    await Promise.all(
      all().map(async ({ state }) => {
        const path = state.view.here.value || deps.opening.value
        if (path) await state.view.go(path)
        await state.reads()
      }),
    )
  }

  return { kind, looking, names, travel, leaves, again }
}
