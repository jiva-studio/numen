import { FolderOpen, FolderTree } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import FilesTab from './ui/FilesTab.vue'
import { createFilesKind, FILES } from './kind'
import { useFileTree } from './model/useFileTree'
import { WORDS } from './words'

export { FilesTab }
export { createFilesKind, FILES } from './kind'
export { createMenuItems, NEW_FOLDER, NEW_NOTE, RENAME } from './lib/menu'
export { ROOT, createFileTree, useFileTree } from './model/useFileTree'
export { useFilesTab } from './model/useFilesTab'
export type { FilesTabDeps, FilesTabState } from './types'
export * from './words'

export function activate(context: PluginContext): void {
  const { workspace, files } = context
  if (!workspace || !files) {
    throw new Error('a files tab needs the window it is drawn in and the files of the vault')
  }
  const trees = createFilesKind({ workspace, files }, () => useFileTree(files))
  context.registerTabKind(trees.kind)
  context.onVaultChange((event) => trees.refresh(event))

  context.registerCommand({
    id: 'files',
    title: WORDS.showFiles,
    icon: FolderTree,
    handler: async () => {
      await context.openTab?.(FILES)
    },
  })
  context.registerCommand({
    id: 'reveal',
    title: WORDS.reveal,
    scope: 'note',
    icon: FolderOpen,
    handler: async (note) => {
      await context.navigateTab?.(FILES, note.path)
    },
  })
}

export default definePlugin({
  activate,
})
