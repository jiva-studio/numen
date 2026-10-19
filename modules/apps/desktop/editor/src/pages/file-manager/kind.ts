/**
 * Window tab registration for the files tab.
 */
import { watch } from 'vue'
import type { PathRename } from '@/shared/paths'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { FILES } from '@/entities/tab'
import FilesTab from './ui/FilesTab.vue'
import { useFilesTab } from './model/useFilesTab'
import { ROOT } from './model/useFileTree'
import { WORDS as words } from './words'
import type { FilesTabDeps, FilesTabState, FileTree } from './types'

export function createFilesKind(
  handle: WindowHandle,
  createTree: () => FileTree,
  deps: FilesTabDeps,
) {
  const kind: TabKind<FilesTabState, typeof FILES> = {
    kind: FILES,
    open: () => {
      const state = useFilesTab(createTree(), deps)
      void state.list.openFolder(ROOT)
      watch(state.list.selectedPaths, (paths) => {
        const path = paths[0]
        if (path) {
          deps.onSelectPath?.(path, state)
        }
      })
      return state
    },
    getTitle: () => words.files,
    pane: FilesTab,
    identity: () => FILES,
    onClose: (state) => {
      state.list.close()
      return true
    },
    navigate: (state, path) => state.list.revealPath(path),
    isTool: true,
    placement: { pane: 'aside', order: 1 },
  }

  const revealPath = async (path: string) => {
    const id = await handle.openTab(FILES)
    await handle.getTabState<FilesTabState>(FILES, id)?.list.revealPath(path)
  }

  /** Every files tab takes the change, the ones out of sight included. */
  const refreshChangedPaths = async (
    paths: readonly string[],
    renames: readonly PathRename[] = [],
  ) => {
    await Promise.all(
      handle.each<FilesTabState>(FILES).map((tab) => tab.state.list.refreshChanged(paths, renames)),
    )
  }

  return { kind, revealPath, refreshChangedPaths }
}
