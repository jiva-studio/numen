import {
  definePlugin,
  type CommandNote,
  type HostServices,
  type PluginContext,
} from '@numen/plugin-sdk'
import { shallowRef } from 'vue'
import FilesTab from './ui/FilesTab.vue'
import { createFilesKind, FILES } from './model/kind'
import { createFileTree, readSortBy, ROOT, type SortBy } from './model/useFileTree'
import type { FilesTabState } from './types'

export { FilesTab }
export { createFilesKind, FILES } from './model/kind'
export { ROOT, createFileTree } from './model/useFileTree'
export type { SortBy } from './model/useFileTree'
export { useFilesTab } from './model/useFilesTab'
export type { FilesTabDeps, FilesTabState } from './types'
export * from './words'

export function activate(plugin: PluginContext, host: HostServices): void {
  const { workspace, drops, slots, commands } = host.ui
  const { files } = host.vault
  if (!workspace || !files) {
    throw new Error('a files tab needs the window it is drawn in and the files of the vault')
  }
  const sortBy = shallowRef<SortBy>(readSortBy(plugin.settings.get('sortBy')))
  plugin.settings.onChange('sortBy', (value) => {
    sortBy.value = readSortBy(value)
  })
  const trees = createFilesKind(
    {
      workspace,
      files,
      ...(drops ? { drops } : {}),
      ...(slots ? { slots } : {}),
      ...(commands ? { commands } : {}),
    },
    () => createFileTree(files, { sortBy }),
    FilesTab,
  )
  plugin.register.tabKind(trees.kind)
  host.events.onVaultChange((event) => trees.refresh(event))

  /** The handler of a command over a row of the tab whose menu was opened last. */
  const runOnTab =
    (run: (state: FilesTabState, path: string) => Promise<void> | void) =>
    async (note: CommandNote): Promise<void> => {
      const state = trees.getMenuTab()
      if (state) await run(state, note.path)
    }

  plugin.register.command({
    id: 'tree.open',
    handler: async (note) => {
      await host.tabs.navigate?.(FILES, note.path)
    },
  })
  plugin.register.command({
    id: 'tree.renameFile',
    handler: runOnTab((state, path) => {
      if (path !== ROOT) state.renaming.setPath(path)
    }),
  })
}

export default definePlugin({
  activate,
})
