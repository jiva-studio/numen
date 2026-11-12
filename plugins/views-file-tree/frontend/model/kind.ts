/**
 * The files tab kind: the folders and files of the vault, one tab of them to a
 * window.
 */
import { watch } from 'vue'
import { FolderTree } from '@lucide/vue'
import type { Component } from 'vue'
import type { TabKindRegistration, VaultChangeEvent } from '@numen/plugin-sdk'
import { useFilesTab } from './useFilesTab'
import { ROOT } from './useFileTree'
import { createFilesTabDeps, type FileTreeHost } from './tabDeps'
import { WORDS as words } from '../words'
import type { FilesTabState, FileTree } from '../types'

/** The word the files tab is filed under, and its identity: one to a window. */
export const FILES = 'files'

/**
 * The files kind as the window keeps it, and what a change to the vault does to
 * its tabs. Each tab draws a tree of its own, made by `createTree`, in `component`.
 */
export function createFilesKind(
  host: FileTreeHost,
  createTree: () => FileTree,
  component: Component,
) {
  const tabs = new Map<string, FilesTabState>()
  let menuTab: FilesTabState | null = null

  const kind: TabKindRegistration<FilesTabState> = {
    kind: FILES,
    component,
    title: words.files,
    icon: FolderTree,
    isTool: true,
    placement: { pane: 'aside', order: 1 },
    open: (targetPath, id) => {
      const deps = createFilesTabDeps(host, id)
      const state = useFilesTab(createTree(), deps)
      tabs.set(id, state)
      state.list.folders.openFolder(ROOT).catch(state.reportFailure)
      if (targetPath) state.list.revealPath(targetPath).catch(state.reportFailure)
      watch(state.list.state.selectedPaths, (paths) => {
        const path = paths[0]
        if (path) deps.window.onSelectPath?.(path, state)
      })
      watch(
        state.menu.request,
        (request) => {
          if (request) menuTab = state
        },
        { flush: 'sync' },
      )
      return state
    },
    getTitle: () => words.files,
    onClose: (state, id) => {
      state.list.close()
      tabs.delete(id)
      if (menuTab === state) menuTab = null
      return true
    },
    navigate: async (state, path) => {
      if (path) await state.list.revealPath(path)
    },
  }

  /** Every files tab takes the change, the ones out of sight included. */
  const refresh = async ({ paths = [], renamed = [] }: VaultChangeEvent) => {
    await Promise.all(
      [...tabs.values()].map((state) => state.list.sync.refreshChanged(paths, renamed)),
    )
  }

  /** The tab whose menu was opened last, or the tab opened last; null while none is open. */
  const getMenuTab = (): FilesTabState | null => menuTab ?? [...tabs.values()].at(-1) ?? null

  return { kind, refresh, getMenuTab }
}
