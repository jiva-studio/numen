/**
 * Window registration and tab state for the files tree tab.
 */
import type { PathRename } from '../../shared/core'
import { type FileTree, ROOT } from './listing'
import type { TabKind, WindowHandle } from '../../shared/tabs/windowTabs'
import { FILES } from '../../shared/tabs/workspace'
import FilesTab from './FilesTab.vue'
import { WORDS as words } from './words'
import type { DropPosition, FileMaker, FilesTabDeps, FilesTabState, MenuRequest } from './types'
import { landingOf, useFilesTab } from './open'

export type { DropPosition, FileMaker, FilesTabDeps, FilesTabState, MenuRequest }
export { landingOf, useFilesTab }

/**
 * Manages window-level files tab operations and path reveal.
 */
export function filesKind(handle: WindowHandle, makes: () => FileTree, deps: FilesTabDeps) {
  const kind: TabKind<FilesTabState, typeof FILES> = {
    kind: FILES,
    opens: () => {
      const state = useFilesTab(makes(), deps)
      void state.list.opens(ROOT)
      return state
    },
    called: () => words.files,
    draws: FilesTab,
    identity: () => FILES,
    shuts: (state) => {
      state.list.close()
      return true
    },
  }

  /** The tree of this window, and nothing while it holds none. */
  const front = (): FilesTabState | null => handle.last<FilesTabState>(FILES)?.state ?? null

  /**
   * The tree put in front of the person, walked down to a path. The window that
   * holds none opens one on it.
   */
  const revealPath = async (path: string) => {
    const id = await handle.opens(FILES)
    await handle.holds<FilesTabState>(FILES, id)?.list.reveals(path)
  }

  /** The vault changed, and every open folder a named path sits in is read again. */
  const refreshChangedPaths = (paths: readonly string[], renamed: readonly PathRename[] = []) =>
    front()?.list.changed(paths, renamed) ?? Promise.resolve()

  return { kind, revealPath, refreshChangedPaths }
}
