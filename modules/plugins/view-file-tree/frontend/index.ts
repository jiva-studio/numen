import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import FilesTab from './ui/FilesTab.vue'
import { createFilesKind } from './kind'
import { useFileTree } from './model/useFileTree'

export { FilesTab }
export { createFilesKind, FILES } from './kind'
export {
  createMenuItems,
  NEW_DECK,
  NEW_FOLDER,
  NEW_NOTE,
  NEW_PRESET,
  NEW_STENCIL,
  RENAME,
} from './lib/menu'
export { ROOT, useFileTree } from './model/useFileTree'
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
}

export default definePlugin({
  activate,
})
