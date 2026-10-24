/**
 * The files tab kind: the folders and files of the vault, one tab of them to a
 * window.
 */
import { watch } from 'vue'
import { FolderTree } from '@lucide/vue'
import type { TabKindRegistration, VaultChangeEvent } from '@numen/plugin-sdk'
import FilesTab from './ui/FilesTab.vue'
import { useFilesTab } from './model/useFilesTab'
import { ROOT } from './model/useFileTree'
import { createFilesTabDeps, type FileTreeHost } from './model/tabDeps'
import { WORDS as words } from './words'
import type { FilesTabState, FileTree } from './types'

/** The word the files tab is filed under, and its identity: one to a window. */
export const FILES = 'files'

/**
 * The files kind as the window keeps it, and what a change to the vault does to
 * its tabs. Each tab draws a tree of its own, made by `createTree`.
 */
export function createFilesKind(host: FileTreeHost, createTree: () => FileTree) {
  const tabs = new Map<string, FilesTabState>()

  const kind: TabKindRegistration<FilesTabState> = {
    kind: FILES,
    component: FilesTab,
    title: words.files,
    icon: FolderTree,
    isTool: true,
    placement: { pane: 'aside', order: 1 },
    open: (targetPath, id) => {
      const deps = createFilesTabDeps(host, id)
      const state = useFilesTab(createTree(), deps)
      tabs.set(id, state)
      void state.list.openFolder(ROOT)
      if (targetPath) void state.list.revealPath(targetPath)
      watch(state.list.selectedPaths, (paths) => {
        const path = paths[0]
        if (path) deps.onSelectPath?.(path, state)
      })
      return state
    },
    getTitle: () => words.files,
    identity: () => FILES,
    onClose: (state, id) => {
      state.list.close()
      tabs.delete(id)
      return true
    },
    navigate: (state, path) => state.list.revealPath(path),
  }

  /** Every files tab takes the change, the ones out of sight included. */
  const refresh = async ({ paths = [], renamed = [] }: VaultChangeEvent) => {
    await Promise.all([...tabs.values()].map((state) => state.list.refreshChanged(paths, renamed)))
  }

  return { kind, refresh }
}
