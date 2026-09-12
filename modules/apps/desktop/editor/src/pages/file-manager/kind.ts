/**
 * Window tab registration for the files tab.
 */
import type { PathRename } from '@/shared/paths'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { FILES } from '@/entities/tab'
import FilesTab from './ui/FilesTab.vue'
import { useFilesTab } from './model/useFilesTab'
import { ROOT } from './model/useFileTree'
import { WORDS as words } from './words'
import type { FilesTabDeps, FilesTabState, FileTree } from './types'

export function filesKind(handle: WindowHandle, makes: () => FileTree, deps: FilesTabDeps) {
  const kind: TabKind<FilesTabState, typeof FILES> = {
    kind: FILES,
    opens: () => {
      const state = useFilesTab(makes(), deps)
      void state.list.openFolder(ROOT)
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

  const getFrontState = (): FilesTabState | null => handle.last<FilesTabState>(FILES)?.state ?? null

  const revealPath = async (path: string) => {
    const id = await handle.opens(FILES)
    await handle.holds<FilesTabState>(FILES, id)?.list.revealPath(path)
  }

  const refreshChangedPaths = (paths: readonly string[], renamed: readonly PathRename[] = []) =>
    getFrontState()?.list.refreshChanged(paths, renamed) ?? Promise.resolve()

  return { kind, revealPath, refreshChangedPaths }
}
