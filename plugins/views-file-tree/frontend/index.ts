import { definePlugin, type CommandNote, type PluginContext } from '@numen/plugin-sdk'
import FilesTab from './ui/FilesTab.vue'
import { createFilesKind, FILES } from './model/kind'
import { createFileTree, ROOT } from './model/useFileTree'
import type { FilesTabState } from './types'

export { FilesTab }
export { createFilesKind, FILES } from './model/kind'
export { ROOT, createFileTree } from './model/useFileTree'
export { useFilesTab } from './model/useFilesTab'
export type { FilesTabDeps, FilesTabState } from './types'
export * from './words'

export function activate(context: PluginContext): void {
  const { workspace, drops, slots, commands } = context.ui
  const { files } = context.vault
  if (!workspace || !files) {
    throw new Error('a files tab needs the window it is drawn in and the files of the vault')
  }
  const trees = createFilesKind(
    {
      workspace,
      files,
      ...(drops ? { drops } : {}),
      ...(slots ? { slots } : {}),
      ...(commands ? { commands } : {}),
    },
    () => createFileTree(files),
    FilesTab,
  )
  context.register.tabKind(trees.kind)
  context.events.onVaultChange((event) => trees.refresh(event))

  /** The handler of a command over a row of the tab whose menu was opened last. */
  const runOnTab =
    (run: (state: FilesTabState, path: string) => Promise<void> | void) =>
    async (note: CommandNote): Promise<void> => {
      const state = trees.getMenuTab()
      if (state) await run(state, note.path)
    }

  context.register.command({
    id: 'tree.open',
    handler: async (note) => {
      await context.tabs.navigate?.(FILES, note.path)
    },
  })
  context.register.command({
    id: 'tree.renameFile',
    handler: runOnTab((state, path) => {
      if (path !== ROOT) state.renaming.setPath(path)
    }),
  })
}

export default definePlugin({
  activate,
})
