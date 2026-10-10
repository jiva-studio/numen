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
  }

  const getFrontState = (): FilesTabState | null => handle.last<FilesTabState>(FILES)?.state ?? null

  const revealPath = async (path: string) => {
    const id = await handle.openTab(FILES)
    await handle.getTabState<FilesTabState>(FILES, id)?.list.revealPath(path)
  }

  const refreshChangedPaths = (paths: readonly string[], renames: readonly PathRename[] = []) =>
    getFrontState()?.list.refreshChanged(paths, renames) ?? Promise.resolve()

  return { kind, revealPath, refreshChangedPaths }
}
